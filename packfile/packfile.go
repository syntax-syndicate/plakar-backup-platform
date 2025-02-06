package packfile

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
)

const VERSION = "1.0.0"

func init() {
	versioning.Register(resources.RT_PACKFILE, versioning.FromString(VERSION))
	versioning.Register(resources.RT_PACKFILE_INDEX, versioning.FromString(VERSION))
	versioning.Register(resources.RT_PACKFILE_FOOTER, versioning.FromString(VERSION))
}

type Blob struct {
	Type     resources.Type
	Version  versioning.Version
	Checksum objects.Checksum
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
	IndexChecksum objects.Checksum
}

type Configuration struct {
	MinSize uint64
	AvgSize uint64
	MaxSize uint64
}

func DefaultConfiguration() *Configuration {
	return &Configuration{
		MaxSize: (20 << 10) << 10,
	}
}

func NewFooterFromBytes(serialized []byte) (PackFileFooter, error) {
	var footer PackFileFooter

	reader := bytes.NewReader(serialized)
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
		var resourceType resources.Type
		var resourceVersion versioning.Version
		var checksum objects.Checksum
		var blobOffset uint32
		var blobLength uint32

		if err := binary.Read(reader, binary.LittleEndian, &resourceType); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &resourceVersion); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &checksum); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobLength); err != nil {
			return nil, err
		}
		index = append(index, Blob{
			Type:     resourceType,
			Version:  resourceVersion,
			Checksum: checksum,
			Offset:   blobOffset,
			Length:   blobLength,
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
		var resourceType resources.Type
		var resourceVersion versioning.Version
		var checksum objects.Checksum
		var blobOffset uint32
		var blobLength uint32

		if err := binary.Read(reader, binary.LittleEndian, &resourceType); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &resourceVersion); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &checksum); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobLength); err != nil {
			return nil, err
		}

		if blobOffset+blobLength > p.Footer.IndexOffset {
			return nil, fmt.Errorf("blob offset + blob length exceeds total length of packfile")
		}

		if err := binary.Write(hasher, binary.LittleEndian, resourceType); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, resourceVersion); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blobOffset); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blobLength); err != nil {
			return nil, err
		}
		p.Index = append(p.Index, Blob{
			Type:     resourceType,
			Version:  resourceVersion,
			Checksum: checksum,
			Offset:   blobOffset,
			Length:   blobLength,
		})
		remaining -= (4 + 4 + len(checksum) + 4 + 4)
	}
	checksum := objects.Checksum(hasher.Sum(nil))
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
	for _, blob := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}

		if err := binary.Write(hasher, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
	}
	p.Footer.IndexChecksum = objects.Checksum(hasher.Sum(nil))

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
	for _, blob := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}

		if err := binary.Write(hasher, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func (p *PackFile) SerializeFooter() ([]byte, error) {
	var buffer bytes.Buffer
	hasher := sha256.New()
	for _, blob := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}

		if err := binary.Write(hasher, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Checksum); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(hasher, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
	}
	p.Footer.IndexChecksum = objects.Checksum(hasher.Sum(nil))

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

func (p *PackFile) AddBlob(resourceType resources.Type, version versioning.Version, checksum objects.Checksum, data []byte) {
	p.Index = append(p.Index, Blob{
		Type:     resourceType,
		Version:  version,
		Checksum: checksum,
		Offset:   uint32(len(p.Blobs)),
		Length:   uint32(len(data)),
	})
	p.Blobs = append(p.Blobs, data...)
	p.Footer.Count++
	p.Footer.IndexOffset = uint32(len(p.Blobs))
}

func (p *PackFile) GetBlob(checksum objects.Checksum) ([]byte, bool) {
	for _, blob := range p.Index {
		if blob.Checksum == checksum {
			return p.Blobs[blob.Offset : blob.Offset+blob.Length], true
		}
	}
	return nil, false
}

func (p *PackFile) Size() uint32 {
	return uint32(len(p.Blobs))
}
