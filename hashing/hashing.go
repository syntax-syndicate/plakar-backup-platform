package hashing

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"hash"

	"github.com/zeebo/blake3"
)

type Configuration struct {
	Algorithm string // Hashing algorithm name (e.g., "SHA256", "BLAKE3")
	Bits      uint32
}

func DefaultConfiguration() *Configuration {
	configuration, _ := LookupDefaultConfiguration("SHA256")
	//configuration, _ := LookupDefaultConfiguration("BLAKE3")
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

func GetHasherHMAC(name string, secret []byte) hash.Hash {
	switch name {
	case "SHA256":
		return hmac.New(sha256.New, secret)
	case "BLAKE3":
		return hmac.New(func() hash.Hash { return blake3.New() }, secret)
	default:
		return nil
	}
}
