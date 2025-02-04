package packfile

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
)

const VERSION = "1.0.0"

type Blob struct {
	Type     resources.Type
	Checksum [32]byte
	Offset   uint32
	Length   uint32
}

type PackFile struct {
	Blobs  []byte
	Index  []Blob
	Footer PackFileFooter
}

type PackFileFooter struct {
	Version       versioning.Version
	Timestamp     int64
	Count         uint32
	IndexOffset   uint32
	IndexChecksum [32]byte
}

type Configuration struct {
	MaxSize uint32
}

func DefaultConfiguration() *Configuration {
	return &Configuration{
		MaxSize: (20 << 10) << 10,
	}
}

func NewFooterFromBytes(serialized []byte) (PackFileFooter, error) {

	reader := bytes.NewReader(serialized)
	var footer PackFileFooter
	if err := binary.Read(reader, binary.LittleEndian, &footer.Version); err != nil {
		return footer, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Timestamp); err != nil {
		return footer, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Count); err != nil {
		return footer, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.IndexOffset); err != nil {
		return footer, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.IndexChecksum); err != nil {
		return footer, err
	}
	return footer, nil
}

func NewIndexFromBytes(serialized []byte) ([]Blob, error) {
	reader := bytes.NewReader(serialized)
	index := make([]Blob, 0)
	for reader.Len() > 0 {
		var dataType resources.Type
		var checksum [32]byte
		var chunkOffset uint32
		var chunkLength uint32
		if err := binary.Read(reader, binary.LittleEndian, &dataType); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &checksum); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &chunkOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &chunkLength); err != nil {
			return nil, err
		}
		index = append(index, Blob{
			Type:     resources.Type(dataType),
			Checksum: checksum,
			Offset:   chunkOffset,
			Length:   chunkLength,
		})
	}
	return index, nil
}

func New() *PackFile {
	return &PackFile{
		Blobs: make([]byte, 0),
		Index: make([]Blob, 0),
		Footer: PackFileFooter{
			Version:   versioning.FromString(VERSION),
			Timestamp: time.Now().UnixNano(),
			Count:     0,
		},
	}
}

func NewFromBytes(serialized []byte) (*PackFile, error) {
	reader := bytes.NewReader(serialized)
	var footer PackFileFooter
	_, err := reader.Seek(-52, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Count); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.IndexOffset); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.IndexChecksum); err != nil {
		return nil, err
	}

	_, err = reader.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	data := make([]byte, footer.IndexOffset)
	if err := binary.Read(reader, binary.LittleEndian, &data); err != nil {
		return nil, err
	}

	// we won't read the totalLength again
	remaining := reader.Len() - 52

	p := New()
	p.Footer = footer
	p.Blobs = data
	hasher := sha256.New()
	for remaining > 0 {
		var dataType resources.Type
		var checksum [32]byte
		var chunkOffset uint32
		var chunkLength uint32
		if err := binary.Read(reader, binary.LittleEndian, &dataType); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &checksum); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &chunkOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &chunkLength); err != nil {
			return nil, err
		}

		if chunkOffset+chunkLength > p.Footer.IndexOffset {
			return nil, fmt.Errorf("chunk offset + chunk length exceeds total length of packfile")
		}

		if err := binary.Write(hasher, binary.LittleEndian, dataType); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunkOffset); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunkLength); err != nil {
			return nil, err
		}
		p.Index = append(p.Index, Blob{
			Type:     dataType,
			Checksum: checksum,
			Offset:   chunkOffset,
			Length:   chunkLength,
		})
		remaining -= (len(checksum) + 8 + 4)
	}
	checksum := [32]byte(hasher.Sum(nil))
	if checksum != p.Footer.IndexChecksum {
		return nil, fmt.Errorf("index checksum mismatch")
	}

	return p, nil
}

func (p *PackFile) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.LittleEndian, p.Blobs); err != nil {
		return nil, err
	}

	hasher := sha256.New()
	for _, chunk := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Length); err != nil {
			return nil, err
		}

		if err := binary.Write(hasher, binary.LittleEndian, chunk.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Length); err != nil {
			return nil, err
		}
	}
	p.Footer.IndexChecksum = [32]byte(hasher.Sum(nil))

	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Version); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Count); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.IndexOffset); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.IndexChecksum); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (p *PackFile) SerializeData() ([]byte, error) {
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.LittleEndian, p.Blobs); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (p *PackFile) SerializeIndex() ([]byte, error) {
	var buffer bytes.Buffer
	hasher := sha256.New()
	for _, chunk := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Length); err != nil {
			return nil, err
		}

		if err := binary.Write(hasher, binary.LittleEndian, chunk.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Length); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func (p *PackFile) SerializeFooter() ([]byte, error) {
	var buffer bytes.Buffer
	hasher := sha256.New()
	for _, chunk := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, chunk.Length); err != nil {
			return nil, err
		}

		if err := binary.Write(hasher, binary.LittleEndian, chunk.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, chunk.Length); err != nil {
			return nil, err
		}
	}
	p.Footer.IndexChecksum = [32]byte(hasher.Sum(nil))

	buffer.Reset()
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Version); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Count); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.IndexOffset); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.IndexChecksum); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (p *PackFile) AddBlob(dataType resources.Type, checksum [32]byte, data []byte) {
	p.Index = append(p.Index, Blob{dataType, checksum, uint32(len(p.Blobs)), uint32(len(data))})
	p.Blobs = append(p.Blobs, data...)
	p.Footer.Count++
	p.Footer.IndexOffset = uint32(len(p.Blobs))
}

func (p *PackFile) GetBlob(checksum [32]byte) ([]byte, bool) {
	for _, chunk := range p.Index {
		if chunk.Checksum == checksum {
			return p.Blobs[chunk.Offset : chunk.Offset+chunk.Length], true
		}
	}
	return nil, false
}

func (p *PackFile) Size() uint32 {
	return uint32(len(p.Blobs))
}
