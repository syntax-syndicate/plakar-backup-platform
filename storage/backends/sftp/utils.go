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

package sftp

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/pkg/sftp"
)

type ClosingFileReader struct {
	reader io.Reader
	file   *sftp.File
}

func (cr *ClosingFileReader) Read(p []byte) (int, error) {
	n, err := cr.reader.Read(p)
	if err == io.EOF {
		// Close the file when EOF is reached
		closeErr := cr.file.Close()
		if closeErr != nil {
			return n, fmt.Errorf("error closing file: %w", closeErr)
		}
	}
	return n, err
}

func ClosingReader(file *sftp.File) (io.Reader, error) {
	return &ClosingFileReader{
		reader: file,
		file:   file,
	}, nil
}

func ClosingLimitedReaderFromOffset(file *sftp.File, offset, length int64) (io.Reader, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	st, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if st.Size() == 0 {
		return bytes.NewBuffer([]byte{}), nil
	}

	if length > (st.Size() - offset) {
		return nil, fmt.Errorf("invalid length")
	}

	return &ClosingFileReader{
		reader: &io.LimitedReader{
			R: file,
			N: length,
		},
		file: file,
	}, nil
}

func WriteToFileAtomic(sftpClient *sftp.Client, filename string, rd io.Reader) error {
	return WriteToFileAtomicTempDir(sftpClient, filename, rd, filepath.Dir(filename))
}

func WriteToFileAtomicTempDir(sftpClient *sftp.Client, filename string, rd io.Reader, tmpdir string) error {
	tmp := fmt.Sprintf("%s.tmp", filename)
	f, err := sftpClient.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, rd); err != nil {
		f.Close()
		sftpClient.Remove(f.Name())
		return err
	}

	if err = f.Close(); err != nil {
		sftpClient.Remove(f.Name())
		return err
	}

	err = sftpClient.Rename(f.Name(), filename)
	if err != nil {
		sftpClient.Remove(f.Name())
		return err
	}

	return nil
}
