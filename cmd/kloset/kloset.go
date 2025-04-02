package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"os"

	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/google/uuid"
)

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
