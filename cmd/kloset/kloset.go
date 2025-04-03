package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"math"
	"os"

	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/google/uuid"
)

// File format:
// [0:8]: CONFIG LEN
// [8:8+CONFIG_LEN]: CONFIG
// [8+CONFIG_LEN:]: PACKFILE * N
// STATE

var currentOffset = uint64(0)
var packfileIdx []packfile.Blob
var packfileFooter packfile.PackFileFooter

var packfileWriter io.Writer = nil

func PutBlob(Type resources.Type, version versioning.Version, mac objects.MAC, data []byte, flags uint32) error {
	_, err := packfileWriter.Write(data)
	if err != nil {
		return err
	}

	packfileIdx = append(packfileIdx, packfile.Blob{
		Type:    Type,
		Version: version,
		MAC:     mac,
		Offset:  currentOffset,
		Length:  uint32(len(data)),
		Flags:   flags,
	})
	currentOffset += uint64(len(data))

	packfileFooter.Count++
	packfileFooter.IndexOffset = currentOffset

	return nil
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatal("No command specified")
	}

	fp, err := os.Create(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	config := storage.NewConfiguration()
	config.RepositoryID = uuid.Must(uuid.NewRandom())
	config.Compression = compression.NewDefaultConfiguration()
	config.Hashing = *hashing.NewDefaultConfiguration()
	config.Encryption = nil
	config.Packfile.MaxSize = math.MaxUint64

	serializedConfig, err := config.ToBytes()
	if err != nil {
		log.Fatal(err)
	}

	config.Encryption = nil
	hasher := hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)

	rd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serializedConfig))
	if err != nil {
		log.Fatal(err)
	}
	wrappedConfig, err := io.ReadAll(rd)
	if err != nil {
		log.Fatal(err)
	}

	headerLength := uint64(len(wrappedConfig))
	binary.Write(fp, binary.LittleEndian, headerLength)
	fp.Write(wrappedConfig)

}
