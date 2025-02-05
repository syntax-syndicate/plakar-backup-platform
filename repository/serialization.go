package repository

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
)

const (
	PHASE_HEADER = iota
	PHASE_DATA
	PHASE_FOOTER
)

func (r *Repository) decode(input io.Reader) (io.Reader, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Decode: %s", time.Since(t0))
	}()

	stream := input
	if r.secret != nil {
		tmp, err := encryption.DecryptStream(r.secret, stream)
		if err != nil {
			return nil, err
		}
		stream = tmp
	}

	if r.configuration.Compression != nil {
		tmp, err := compression.InflateStream(r.configuration.Compression.Algorithm, stream)
		if err != nil {
			return nil, err
		}
		stream = tmp
	}

	return stream, nil
}

func (r *Repository) encode(input io.Reader) (io.Reader, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Encode: %s", time.Since(t0))
	}()

	stream := input
	if r.configuration.Compression != nil {
		tmp, err := compression.DeflateStream(r.configuration.Compression.Algorithm, stream)
		if err != nil {
			return nil, err
		}
		stream = tmp
	}

	if r.secret != nil {
		tmp, err := encryption.EncryptStream(r.secret, stream)
		if err != nil {
			return nil, err
		}
		stream = tmp
	}

	return stream, nil
}

func (r *Repository) DeserializeBuffer(buffer []byte) ([]byte, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "DeserializeBuffer(%d bytes): %s", len(buffer), time.Since(t0))
	}()

	rd, err := r.Deserialize(bytes.NewBuffer(buffer))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(rd)
}

func (r *Repository) SerializeBuffer(resourceType resources.Type, version versioning.Version, buffer []byte) ([]byte, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "SerializeBuffer(%d): %s", len(buffer), time.Since(t0))
	}()

	rd, err := r.Serialize(resourceType, version, bytes.NewBuffer(buffer))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(rd)
}

type deserializeReader struct {
	inner    io.Reader
	leftOver []byte

	hasher hash.Hash
	hmac   [32]byte

	eof bool
}

func (r *Repository) newDeserializeReader(inner io.Reader) (*deserializeReader, error) {
	buf := make([]byte, 8)
	_, err := io.ReadFull(inner, buf)
	if err != nil {
		return nil, err
	}

	hasher := r.HMAC()
	hasher.Write(buf)

	return &deserializeReader{
		inner:  inner,
		hasher: hasher,
		hmac:   [32]byte{},
	}, nil
}

func (s *deserializeReader) Read(p []byte) (int, error) {
	total := 0

	for total < len(p) {
		// 1. If not at EOF, read a chunk from the inner reader.
		if !s.eof {
			const chunkSize = 4096
			buf := make([]byte, chunkSize)
			n, err := s.inner.Read(buf)
			if n > 0 {
				s.leftOver = append(s.leftOver, buf[:n]...)
			}
			if err != nil {
				if err == io.EOF {
					s.eof = true
				} else {
					return total, err
				}
			}
		}

		flushable := len(s.leftOver) - 32
		if flushable < 0 {
			flushable = 0
		}

		avail := len(p) - total
		if flushable > 0 {
			nFlush := flushable
			if nFlush > avail {
				nFlush = avail
			}
			copy(p[total:total+nFlush], s.leftOver[:nFlush])
			s.hasher.Write(s.leftOver[:nFlush])
			total += nFlush
			s.leftOver = s.leftOver[nFlush:]
		}

		if s.eof && len(s.leftOver) == 32 {
			copy(s.hmac[:], s.leftOver)

			//s.hmac = [32]byte{}

			//s.innerChecksum = [32]byte{}
			//s.outerChecksum = [32]byte{}

			if !bytes.Equal(s.hmac[:], s.hasher.Sum(nil)) {
				return 0, fmt.Errorf("hmac mismatch")
			}
			return total, io.EOF
		}

		if flushable == 0 && !s.eof {
			continue
		}

		if total == len(p) {
			break
		}
	}

	return total, nil
}

func (r *Repository) Deserialize(input io.Reader) (io.Reader, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Deserialize: %s", time.Since(t0))
	}()

	rd, err := r.newDeserializeReader(input)
	if err != nil {
		return nil, err
	}
	return r.decode(rd)
}

type serializeReader struct {
	header       []byte
	headerOffset int

	inner io.Reader

	hasher     hash.Hash
	hmac       [32]byte
	hmacOffset int

	phase int
}

func (r *Repository) newSerializeReader(resourceType resources.Type, version versioning.Version, inner io.Reader) *serializeReader {
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[4:8], uint32(version))
	binary.LittleEndian.PutUint32(header[0:4], uint32(resourceType))

	hasher := r.HMAC()
	hasher.Write(header)

	return &serializeReader{
		header: header,
		inner:  inner,
		hasher: hasher,
		phase:  PHASE_HEADER,
		hmac:   [32]byte{},
	}
}

func (s *serializeReader) Read(p []byte) (n int, err error) {
	total := 0

	for total < len(p) {
		switch s.phase {
		case PHASE_HEADER:
			if s.headerOffset < len(s.header) {
				n = copy(p[total:], s.header[s.headerOffset:])
				s.headerOffset += n
				total += n

				// we filled p, let's do another round !
				if total == len(p) {
					return total, nil
				}
			}
			s.phase = PHASE_DATA

		case PHASE_DATA:
			n, err := s.inner.Read(p[total:])

			// can return err WITH data because EOF
			if n > 0 {
				s.hasher.Write(p[total : total+n])
				total += n
			}

			// if error and not EOF, we return err
			// if EOF, we move to the footer phase
			// if no error we return partial buffer
			if err != nil && err != io.EOF {
				return total, err
			} else {
				if err == io.EOF {
					s.phase = PHASE_FOOTER
					copy(s.hmac[:32], s.hasher.Sum(nil))
					continue
				}
				return total, nil
			}

		case PHASE_FOOTER:
			if s.hmacOffset < len(s.hmac) {
				n = copy(p[total:], s.hmac[s.hmacOffset:])
				s.hmacOffset += n
				total += n
			}
			if s.hmacOffset == len(s.hmac) {
				return total, io.EOF
			} else {
				return total, nil
			}

		default:
			panic("invalid phase, logic error")
		}
	}

	return total, nil
}

func (r *Repository) Serialize(resourceType resources.Type, version versioning.Version, input io.Reader) (io.Reader, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Serialize: %s", time.Since(t0))
	}()

	encoded, err := r.encode(input)
	if err != nil {
		return nil, err
	}

	fmt.Println("serialize", resourceType, version)

	return r.newSerializeReader(resourceType, version, encoded), nil
}
