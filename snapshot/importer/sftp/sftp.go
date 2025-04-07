/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package sftp

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/PlakarKorp/plakar/snapshot/importer"
	sshcfg "github.com/kevinburke/ssh_config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

type SFTPImporter struct {
	rootDir    string
	remoteHost string
	client     *sftp.Client
}

func init() {
	importer.Register("sftp", NewSFTPImporter)
}

func resolveSSHConfig(username string, host string, port string) map[string]string {
	cfgPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	file, err := os.Open(cfgPath)
	if err != nil {
		return map[string]string{}
	}
	defer file.Close()

	cfg, err := sshcfg.Decode(file)
	if err != nil {
		return map[string]string{}
	}

	get := func(field string) string {
		val, err := cfg.Get(host, field)
		if err != nil {
			return ""
		}
		return val
	}

	return map[string]string{
		"host":                 fallback(get("HostName"), host),
		"user":                 fallback(get("User"), username),
		"port":                 fallback(get("Port"), "22"),
		"identity":             get("IdentityFile"),
		"known_hosts":          get("UserKnownHostsFile"),
		"strict_host_checking": get("StrictHostKeyChecking"),
		"proxy_command":        get("ProxyCommand"),
	}
}

func fallback(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func loadSignersForHost(host string, keyPath string) ([]ssh.Signer, error) {
	var signers []ssh.Signer

	// 0. Check for explicitly configured identity file
	if keyPath != "" {
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read specified key %s: %w", keyPath, err)
		}

		// Try without passphrase
		signer, err := ssh.ParsePrivateKey(data)
		if err == nil {
			return []ssh.Signer{signer}, nil
		}

		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			fmt.Printf("Enter passphrase for %s: ", keyPath)
			passphrase, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return nil, fmt.Errorf("failed to read passphrase: %w", err)
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(data, passphrase)
			if err != nil {
				return nil, fmt.Errorf("invalid passphrase for %s: %w", keyPath, err)
			}
			return []ssh.Signer{signer}, nil
		}

		return nil, fmt.Errorf("failed to parse specified key %s: %w", keyPath, err)
	}

	// 1. Check SSH agent
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			ag := agent.NewClient(conn)
			if agentSigners, err := ag.Signers(); err == nil && len(agentSigners) > 0 {
				return agentSigners, nil // ✅ Use agent keys silently
			}
		}
	}

	// 2. Fallback to local keys
	keyFiles := []string{
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ecdsa"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_dsa"),
	}

	for _, file := range keyFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Try without passphrase
		signer, err := ssh.ParsePrivateKey(data)
		if err == nil {
			signers = append(signers, signer)
			continue
		}

		// Prompt if encrypted
		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			fmt.Printf("Enter passphrase for %s: ", file)
			passphrase, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				continue
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(data, passphrase)
			if err == nil {
				signers = append(signers, signer)
			}
		}
	}

	if len(signers) == 0 {
		return nil, fmt.Errorf("no usable SSH keys found")
	}

	return signers, nil
}

func sha256Fingerprint(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
}

func knownHostLine(hostname string, key ssh.PublicKey) string {
	authorized := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))

	// Only format [host]:port if it's a valid host:port
	host, port, err := net.SplitHostPort(hostname)
	if err == nil && port != "22" {
		return fmt.Sprintf("[%s]:%s %s", host, port, authorized)
	}

	// Otherwise, fallback to the raw hostname
	return fmt.Sprintf("%s %s", hostname, authorized)
}

func safeHostKeyCallback(knownHostsPath string, ignore bool) (ssh.HostKeyCallback, error) {
	if ignore {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	if _, err := os.Stat(knownHostsPath); err == nil {
		return knownhosts.New(knownHostsPath)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := sha256Fingerprint(key)
		fmt.Printf("The authenticity of host '%s' can't be established.\n", hostname)
		fmt.Printf("%s key fingerprint is %s.\n", key.Type(), fingerprint)
		fmt.Print("Are you sure you want to continue connecting (yes/no)? ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read user input")
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "yes" {
			return fmt.Errorf("host key not accepted")
		}

		// Append to known_hosts file
		line := knownHostLine(hostname, key)
		if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
			return fmt.Errorf("could not create .ssh dir: %w", err)
		}
		f, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open known_hosts file: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write known_hosts entry: %w", err)
		}

		return nil
	}, nil
}

func connect(endpoint *url.URL, config map[string]string) (*sftp.Client, error) {
	identity := config["identity"]
	ignoreHostKey := config["insecure_ignore_host_key"] == "true"

	sshConfig := resolveSSHConfig(endpoint.User.Username(), endpoint.Hostname(), endpoint.Port())

	var username string
	var host string
	var port string

	host = sshConfig["host"]

	if endpoint.User.Username() != "" {
		username = endpoint.User.Username()
	} else {
		username = sshConfig["user"]
		if username == "" {
			username = os.Getenv("USER")
			if username == "" {
				u, err := user.Current()
				if err != nil {
					return nil, err
				}
				username = u.Username
			}
		}
	}

	if endpoint.Port() != "" {
		port = endpoint.Port()
	} else {
		port = sshConfig["port"]
	}

	if sshConfig["identity"] != "" && identity == "" {
		identity = sshConfig["identity"]
		if strings.HasPrefix(identity, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %v", err)
			}
			identity = filepath.Join(home, identity[2:])
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")
	if customPath := sshConfig["known_hosts"]; customPath != "" {
		knownHostsPath = customPath
	}

	if sshConfig["strict_host_checking"] != "" && !ignoreHostKey {
		switch strings.ToLower(sshConfig["strict_host_checking"]) {
		case "no":
			ignoreHostKey = true
		case "yes":
			ignoreHostKey = false
		case "ask":
			// current prompt behavior
		default:
			// fallback to current prompt behavior
		}
	}

	hostKeyCallback, err := safeHostKeyCallback(knownHostsPath, ignoreHostKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create host key callback: %w", err)
	}

	target := net.JoinHostPort(host, port)
	proxyCommand := sshConfig["proxy_command"]

	var conn net.Conn
	if proxyCommand != "" {
		proxyCommand = strings.ReplaceAll(proxyCommand, "%h", host)
		proxyCommand = strings.ReplaceAll(proxyCommand, "%p", port)

		cmd := exec.Command("sh", "-c", proxyCommand)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("proxy stdin pipe failed: %w", err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("proxy stdout pipe failed: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("proxy command failed to start: %w", err)
		}

		conn = &proxyConn{
			stdin:  stdin,
			stdout: stdout,
			cmd:    cmd,
		}
	} else {
		conn, err = net.Dial("tcp", target)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %w", target, err)
		}
	}
	sshClientConn, chans, reqs, err := ssh.NewClientConn(conn, target, &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
				return loadSignersForHost(host, identity)
			}),
		},
		HostKeyCallback: hostKeyCallback,
	})
	if err != nil {
		return nil, fmt.Errorf("ssh handshake failed: %w", err)
	}

	client := ssh.NewClient(sshClientConn, chans, reqs)

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, err
	}
	return sftpClient, nil
}

func NewSFTPImporter(config map[string]string) (importer.Importer, error) {
	var err error

	location := config["location"]
	if location == "" {
		return nil, fmt.Errorf("missing location")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	location = parsed.Path

	client, err := connect(parsed, config)
	if err != nil {
		return nil, err
	}

	return &SFTPImporter{
		rootDir:    location,
		remoteHost: parsed.Host,
		client:     client,
	}, nil
}

func (p *SFTPImporter) Origin() string {
	return p.remoteHost
}

func (p *SFTPImporter) Type() string {
	return "sftp"
}

func (p *SFTPImporter) Scan() (<-chan *importer.ScanResult, error) {
	return p.walkDir_walker(256)
}

func (p *SFTPImporter) NewReader(pathname string) (io.ReadCloser, error) {
	return p.client.Open(pathname)
}

func (p *SFTPImporter) NewExtendedAttributeReader(pathname string, attribute string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("extended attributes are not supported by the sftp importer yet")
}

func (p *SFTPImporter) GetExtendedAttributes(pathname string) ([]importer.ExtendedAttributes, error) {
	return nil, fmt.Errorf("extended attributes are not supported by the sftp importer yet")
}

func (p *SFTPImporter) Close() error {
	return p.client.Close()
}

func (p *SFTPImporter) Root() string {
	return p.rootDir
}

// wrap stdin/stdout of ProxyCommand as net.Conn

// Wraps stdin/stdout of ProxyCommand as a net.Conn
type proxyConn struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

func (p *proxyConn) Read(b []byte) (int, error)         { return p.stdout.Read(b) }
func (p *proxyConn) Write(b []byte) (int, error)        { return p.stdin.Write(b) }
func (p *proxyConn) Close() error                       { return p.cmd.Process.Kill() }
func (p *proxyConn) LocalAddr() net.Addr                { return dummyAddr("local") }
func (p *proxyConn) RemoteAddr() net.Addr               { return dummyAddr("remote") }
func (p *proxyConn) SetDeadline(t time.Time) error      { return nil }
func (p *proxyConn) SetReadDeadline(t time.Time) error  { return nil }
func (p *proxyConn) SetWriteDeadline(t time.Time) error { return nil }

type dummyAddr string

func (d dummyAddr) Network() string { return string(d) }
func (d dummyAddr) String() string  { return string(d) }
