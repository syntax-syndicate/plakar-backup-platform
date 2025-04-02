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

package kloset

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/storage"
)

type Store struct {
	config     []byte
	Repository string
	location   string
	fp         io.ReadCloser
}

func init() {
	storage.Register(NewStore, "kloset")
}

func NewStore(storeConfig map[string]string) (storage.Store, error) {
	return &Store{
		location: storeConfig["location"],
	}, nil
}

func (s *Store) Location() string {
	return s.location
}

func (s *Store) Create(config []byte) error {
	s.config = config
	return fmt.Errorf("not implemented")
}

func (s *Store) Open() ([]byte, error) {
	location := strings.TrimPrefix(s.location, "kloset://")

	fp, err := os.OpenFile(location, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	s.fp = fp

	lenBuf := make([]byte, 8)
	_, err = io.ReadFull(s.fp, lenBuf)
	if err != nil {
		return nil, err
	}
	headerSize := binary.LittleEndian.Uint64(lenBuf)
	headerBuf := make([]byte, headerSize)
	_, err = io.ReadFull(s.fp, headerBuf)
	if err != nil {
		return nil, err
	}

	return headerBuf, nil
}

func (s *Store) Close() error {
	s.fp.Close()
	return nil
}

func (s *Store) Mode() storage.Mode {
	return storage.ModeRead | storage.ModeWrite
}

// snapshots
func (s *Store) GetSnapshots() ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutSnapshot(snapshotID objects.MAC, data []byte) error {
	return nil
}

func (s *Store) GetSnapshot(snapshotID objects.MAC) ([]byte, error) {
	return []byte{}, nil
}

func (s *Store) DeleteSnapshot(snapshotID objects.MAC) error {
	return fmt.Errorf("not implemented")
}

// states
func (s *Store) GetStates() ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutState(mac objects.MAC, rd io.Reader) error {
	return nil
}

func (s *Store) GetState(mac objects.MAC) (io.Reader, error) {
	return bytes.NewBuffer([]byte{}), nil
}

func (s *Store) DeleteState(mac objects.MAC) error {
	return fmt.Errorf("not implemented")
}

// packfiles
func (s *Store) GetPackfiles() ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutPackfile(mac objects.MAC, rd io.Reader) error {
	return nil
}

func (s *Store) GetPackfile(mac objects.MAC) (io.Reader, error) {
	return bytes.NewBuffer([]byte{}), nil
}

func (s *Store) GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error) {
	return bytes.NewBuffer([]byte{}), nil
}

func (s *Store) DeletePackfile(mac objects.MAC) error {
	return fmt.Errorf("not implemented")
}

/* Locks */
func (s *Store) GetLocks() ([]objects.MAC, error) {
	return []objects.MAC{}, fmt.Errorf("not implemented")
}

func (s *Store) PutLock(lockID objects.MAC, rd io.Reader) error {
	return fmt.Errorf("not implemented")
}

func (s *Store) GetLock(lockID objects.MAC) (io.Reader, error) {
	return bytes.NewBuffer([]byte{}), fmt.Errorf("not implemented")
}

func (s *Store) DeleteLock(lockID objects.MAC) error {
	return fmt.Errorf("not implemented")
}
