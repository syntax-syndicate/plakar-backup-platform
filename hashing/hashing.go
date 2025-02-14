package hashing

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"hash"

	"github.com/zeebo/blake3"
)

const DEFAULT_HASHING_ALGORITHM = "BLAKE3"

type Configuration struct {
	Algorithm string // Hashing algorithm name (e.g., "SHA256", "BLAKE3")
	Bits      uint32
}

func NewDefaultConfiguration() *Configuration {
	configuration, _ := LookupDefaultConfiguration(DEFAULT_HASHING_ALGORITHM)
	return configuration
}

func LookupDefaultConfiguration(algorithm string) (*Configuration, error) {
	switch algorithm {
	case "SHA256":
		return &Configuration{
			Algorithm: "SHA256",
			Bits:      256,
		}, nil
	case "BLAKE3":
		return &Configuration{
			Algorithm: "BLAKE3",
			Bits:      256,
		}, nil
	default:
		return nil, fmt.Errorf("unknown hashing algorithm: %s", algorithm)
	}
}

func GetHasher(name string) hash.Hash {
	switch name {
	case "SHA256":
		return sha256.New()
	case "BLAKE3":
		return blake3.New()
	default:
		return nil
	}
}

func GetMACHasher(name string, secret []byte) hash.Hash {
	switch name {
	case "SHA256":
		return hmac.New(sha256.New, secret)
	case "BLAKE3":
		// secret is guaranteed to be 32 bytes here
		keyed, err := blake3.NewKeyed(secret)
		if err != nil {
			panic(err)
		}
		return keyed
	default:
		return nil
	}
}
