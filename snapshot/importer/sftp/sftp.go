/*
 * Copyright (c) 2023 Gilles Chehade <gilles@poolp.org>
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
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/PlakarKorp/plakar/snapshot/importer"
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

func loadSignersForHost(host string) ([]ssh.Signer, error) {
	var signers []ssh.Signer

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
	return fmt.Sprintf("%s %s", hostname, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
}

func safeHostKeyCallback(knownHostsPath string) (ssh.HostKeyCallback, error) {
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

func connect(endpoint *url.URL) (*sftp.Client, error) {

	var sshHost string
	if endpoint.Port() == "" {
		sshHost = endpoint.Host + ":22"
	} else {
		sshHost = endpoint.Host
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}
	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")

	hostKeyCallback, err := safeHostKeyCallback(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create host key callback: %w", err)
	}

	username := endpoint.User.Username()
	if username == "" {
		u, err := user.Current()
		if err != nil {
			return nil, err
		}
		username = u.Username
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
				return loadSignersForHost(endpoint.Host)
			}),
		},
		HostKeyCallback: hostKeyCallback,
	}

	client, err := ssh.Dial("tcp", sshHost, config)
	if err != nil {
		return nil, err
	}
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

	client, err := connect(parsed)
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
