package testing

import (
	"bytes"
	"io"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/storage"
)

func init() {
	storage.Register("fs", func(location string) storage.Store { return &MockBackend{location: location} })
}

// MockBackend implements the Backend interface for testing purposes
type MockBackend struct {
	configuration storage.Configuration
	location      string
}

func (mb *MockBackend) Create(repository string, configuration storage.Configuration) error {
	mb.configuration = configuration
	return nil
}

func (mb *MockBackend) Open(repository string) error {
	return nil
}

func (mb *MockBackend) Configuration() storage.Configuration {
	return mb.configuration
}

func (mb *MockBackend) Location() string {
	return mb.location
}

func (mb *MockBackend) GetStates() ([]objects.Checksum, error) {
	return nil, nil
}

func (mb *MockBackend) PutState(checksum objects.Checksum, rd io.Reader) error {
	return nil
}

func (mb *MockBackend) GetState(checksum objects.Checksum) (io.Reader, error) {
	return bytes.NewReader([]byte("test data")), nil
}

func (mb *MockBackend) DeleteState(checksum objects.Checksum) error {
	return nil
}

func (mb *MockBackend) GetPackfiles() ([]objects.Checksum, error) {
	return nil, nil
}

func (mb *MockBackend) PutPackfile(checksum objects.Checksum, rd io.Reader) error {
	return nil
}

func (mb *MockBackend) GetPackfile(checksum objects.Checksum) (io.Reader, error) {
	return bytes.NewReader([]byte("packfile data")), nil
}

func (mb *MockBackend) GetPackfileBlob(checksum objects.Checksum, offset uint32, length uint32) (io.Reader, error) {
	return bytes.NewReader([]byte("blob data")), nil
}

func (mb *MockBackend) DeletePackfile(checksum objects.Checksum) error {
	return nil
}

func (mb *MockBackend) Close() error {
	return nil
}
