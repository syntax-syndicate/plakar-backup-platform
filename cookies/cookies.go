package cookies

import (
	"fmt"
	"os"
	"path/filepath"
)

const COOKIES_VERSION = "1.0.0"

type Manager struct {
	cookiesDir string
}

func NewManager(cookiesDir string) *Manager {
	cookiesDir = filepath.Join(cookiesDir, "cookies", COOKIES_VERSION)
	if err := os.MkdirAll(cookiesDir, 0700); err != nil {
		panic(fmt.Errorf("cannot create cookies directory: %w", err))
	}
	if err := os.Chmod(cookiesDir, 0700); err != nil {
		panic(fmt.Errorf("cannot set permissions for cookies directory: %w", err))
	}
	return &Manager{
		cookiesDir: cookiesDir,
	}
}

func (m *Manager) Close() error {
	return nil
}

func (c *Manager) GetAuthToken() (string, error) {
	data, err := os.ReadFile(filepath.Join(c.cookiesDir, ".auth-token"))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("no auth token found")
	}
	return string(data), nil
}

func (c *Manager) HasAuthToken() bool {
	_, err := os.Stat(filepath.Join(c.cookiesDir, ".auth-token"))
	return err == nil
}

func (c *Manager) DeleteAuthToken() error {
	return os.Remove(filepath.Join(c.cookiesDir, ".auth-token"))
}

func (c *Manager) PutAuthToken(token string) error {
	return os.WriteFile(filepath.Join(c.cookiesDir, ".auth-token"), []byte(token), 0600)
}
