/*
 * Copyright (c) 2025 Eric Faurot <eric@faurot.net>
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

package fs

import (
	"bytes"
	"path/filepath"
	"fmt"
	"io"
	"os"
)


type LimitedReaderWithClose struct {
	*io.LimitedReader
	file *os.File
}

func (l *LimitedReaderWithClose) Read(p []byte) (int, error) {
	n, err := l.LimitedReader.Read(p)
	if err == io.EOF {
		// Close the file when EOF is reached
		closeErr := l.file.Close()
		if closeErr != nil {
			return n, fmt.Errorf("error closing file: %w", closeErr)
		}
	}
	return n, err
}


func SliceReader(file *os.File, offset uint32, length uint32) (io.Reader, error) {
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, err
	}

	st, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if st.Size() == 0 {
		return bytes.NewBuffer([]byte{}), nil
	}

	if length > (uint32(st.Size()) - offset) {
		return nil, fmt.Errorf("invalid length")
	}

	return &LimitedReaderWithClose{
		LimitedReader: &io.LimitedReader{
			R: file,
			N: int64(length),
		},
		file: file,
	}, nil
}


func WriteToFileAtomic(filename string, rd io.Reader) error {
	return WriteToFileAtomicTempDir(filename, rd, filepath.Dir(filename))
}


func WriteToFileAtomicTempDir(filename string, rd io.Reader, tmpdir string) error {
	f, err := os.CreateTemp(tmpdir, "tmp.")
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, rd); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}

	if err = f.Close(); err != nil {
		os.Remove(f.Name())
		return err
	}

	err = os.Rename(f.Name(), filename)
	if err != nil {
		os.Remove(f.Name())
		return err
	}

	return nil
}
