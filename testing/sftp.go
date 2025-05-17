package testing

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// MockSFTPServer implements a simple SFTP server for testing
type MockSFTPServer struct {
	Addr     string
	config   *ssh.ServerConfig
	rootDir  string
	pubKey   ssh.PublicKey
	KeyFile  string
	listener net.Listener
}

func NewMockSFTPServer(t *testing.T) (*MockSFTPServer, error) {
	// Generate a keypair for the test
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	pubKey, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %v", err)
	}

	// Write private key to a temp file
	keyFile, err := os.CreateTemp("", "sftp-exporter-key-*.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp key file: %v", err)
	}
	t.Cleanup(func() { os.Remove(keyFile.Name()) })
	if _, err := keyFile.Write(privPEM); err != nil {
		return nil, fmt.Errorf("failed to write private key: %v", err)
	}
	keyFile.Close()

	// Create a temporary directory for the server
	rootDir, err := os.MkdirTemp("", "sftp-server-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if pubKey != nil && bytes.Equal(key.Marshal(), pubKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		os.RemoveAll(rootDir)
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	signer, err := ssh.ParsePrivateKey(privateKeyPEM)
	if err != nil {
		os.RemoveAll(rootDir)
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.RemoveAll(rootDir)
		return nil, err
	}

	server := &MockSFTPServer{
		Addr:     listener.Addr().String(),
		config:   config,
		rootDir:  rootDir,
		pubKey:   pubKey,
		KeyFile:  keyFile.Name(),
		listener: listener,
	}
	go server.serve(listener)
	return server, nil
}

func (s *MockSFTPServer) serve(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *MockSFTPServer) handleConnection(conn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "sftp" {
						req.Reply(true, nil)
						s.handleSFTP(channel)
					}
				}
			}
		}(requests)
	}
}

func (s *MockSFTPServer) handleSFTP(channel ssh.Channel) {
	// Save the current working directory
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	// Change to the temp root directory
	os.Chdir(s.rootDir)

	server, err := sftp.NewServer(channel)
	if err != nil {
		return
	}
	defer server.Close()
	server.Serve()
}

func (s *MockSFTPServer) Close() {
	if s.listener != nil {
		s.listener.Close() // Close the listener to unblock Accept()
	}
	os.RemoveAll(s.rootDir)
}
