/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
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
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Store struct {
	location  string
	packfiles Buckets
	states    Buckets
	client    *sftp.Client
}

func init() {
	storage.Register(NewStore, "sftp")
}

func defaultSigners() ([]ssh.Signer, error) {
	var signers []ssh.Signer

	// Try the SSH agent first.
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		conn, err := net.Dial("unix", sock)
		if err == nil {
			ag := agent.NewClient(conn)
			if s, err := ag.Signers(); err == nil {
				signers = append(signers, s...)
			}
		}
	}

	// Fallback: load from default key files.
	home, err := os.UserHomeDir()
	if err != nil {
		return signers, err
	}

	// List of default private key paths.
	keyFiles := []string{
		path.Join(home, ".ssh", "id_rsa"),
		path.Join(home, ".ssh", "id_dsa"),
		path.Join(home, ".ssh", "id_ecdsa"),
		path.Join(home, ".ssh", "id_ed25519"),
	}

	for _, file := range keyFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that don't exist.
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			continue // Skip unparsable keys.
		}
		signers = append(signers, signer)
	}

	return signers, nil
}

func NewStore(storeConfig map[string]string) (storage.Store, error) {
	return &Store{
		location: storeConfig["location"],
	}, nil
}

func (s *Store) Location() string {
	return s.location
}

func (s *Store) Path(args ...string) string {
	root := s.Location()
	if strings.HasPrefix(root, "sftp://") {
		root = root[7:]
	}
	atoms := strings.Split(root, "/")
	if len(atoms) == 0 {
		return "/"
	} else {
		root = "/" + strings.Join(atoms[1:], "/")
	}

	args = append(args, "")
	copy(args[1:], args)
	args[0] = root

	return path.Join(args...)
}

func connect(location string) (*sftp.Client, error) {
	parsed, err := url.Parse(location)
	if err != nil {
		return nil, err
	}

	var sshHost string
	if parsed.Port() == "" {
		sshHost = parsed.Host + ":22"
	} else {
		sshHost = parsed.Host
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}
	knownHostsPath := path.Join(homeDir, ".ssh", "known_hosts")

	// Create the HostKeyCallback from the known_hosts file.
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("could not create hostkeycallback function: %v", err)
	}

	signers, err := defaultSigners()
	if err != nil {
		return nil, err
	}

	username := parsed.User.Username()
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
			ssh.PublicKeys(signers...),
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

func (s *Store) Create(config []byte) error {
	client, err := connect(s.location)
	if err != nil {
		return err
	}
	s.client = client

	dirfp, err := client.ReadDir(s.Path())
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		err = client.MkdirAll(s.Path())
		if err != nil {
			return err
		}
		err = client.Chmod(s.Path(), 0700)
		if err != nil {
			return err
		}
	} else {
		if len(dirfp) > 0 {
			return fmt.Errorf("directory %s is not empty", s.Location())
		}
	}
	s.packfiles = NewBuckets(client, s.Path("packfiles"))
	if err := s.packfiles.Create(); err != nil {
		return err
	}

	s.states = NewBuckets(client, s.Path("states"))
	if err := s.states.Create(); err != nil {
		return err
	}

	err = client.Mkdir(s.Path("locks"))
	if err != nil {
		return err
	}

	return WriteToFileAtomic(client, s.Path("CONFIG"), bytes.NewReader(config))
}

func (s *Store) Open() ([]byte, error) {
	client, err := connect(s.location)
	if err != nil {
		return nil, err
	}
	s.client = client

	rd, err := client.Open(s.Path("CONFIG"))
	if err != nil {
		return nil, err
	}
	defer rd.Close() // do we care about err?

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	s.packfiles = NewBuckets(client, s.Path("packfiles"))

	s.states = NewBuckets(client, s.Path("states"))

	return data, nil
}

func (s *Store) GetPackfiles() ([]objects.MAC, error) {
	return s.packfiles.List()
}

func (s *Store) GetPackfile(mac objects.MAC) (io.Reader, error) {
	fp, err := s.packfiles.Get(mac)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = repository.ErrPackfileNotFound
		}
		return nil, err
	}

	return fp, nil
}

func (s *Store) GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error) {
	res, err := s.packfiles.GetBlob(mac, offset, length)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = repository.ErrPackfileNotFound
		}
		return nil, err
	}
	return res, nil
}

func (s *Store) Mode() storage.Mode {
	return storage.ModeRead | storage.ModeWrite
}

func (s *Store) DeletePackfile(mac objects.MAC) error {
	return s.packfiles.Remove(mac)
}

func (s *Store) PutPackfile(mac objects.MAC, rd io.Reader) error {
	return s.packfiles.Put(mac, rd)
}

func (s *Store) Close() error {
	return nil
}

/* Indexes */
func (s *Store) GetStates() ([]objects.MAC, error) {
	return s.states.List()
}

func (s *Store) PutState(mac objects.MAC, rd io.Reader) error {
	return s.states.Put(mac, rd)
}

func (s *Store) GetState(mac objects.MAC) (io.Reader, error) {
	return s.states.Get(mac)
}

func (s *Store) DeleteState(mac objects.MAC) error {
	return s.states.Remove(mac)
}

/* Locks */
func (s *Store) GetLocks() (ret []objects.MAC, err error) {
	entries, err := s.client.ReadDir(s.Path("locks"))
	if err != nil {
		return
	}

	for i := range entries {
		var t []byte
		t, err = hex.DecodeString(entries[i].Name())
		if err != nil {
			return
		}
		if len(t) != 32 {
			continue
		}
		ret = append(ret, objects.MAC(t))
	}
	return
}

func (s *Store) PutLock(lockID objects.MAC, rd io.Reader) error {
	return WriteToFileAtomicTempDir(s.client, filepath.Join(s.Path("locks"), hex.EncodeToString(lockID[:])), rd, s.Path(""))
}

func (s *Store) GetLock(lockID objects.MAC) (io.Reader, error) {
	fp, err := s.client.Open(filepath.Join(s.Path("locks"), hex.EncodeToString(lockID[:])))
	if err != nil {
		return nil, err
	}

	return ClosingReader(fp)
}

func (s *Store) DeleteLock(lockID objects.MAC) error {
	return s.client.Remove(filepath.Join(s.Path("locks"), hex.EncodeToString(lockID[:])))
}
