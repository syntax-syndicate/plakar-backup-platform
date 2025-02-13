package hashing

import (
	"testing"
)

func TestDefaultAlgorithm(t *testing.T) {
	expected := "BLAKE3"
	actual := NewDefaultConfiguration().Algorithm
	if actual != expected {
		t.Errorf("Expected default algorithm %s, but got %s", expected, actual)
	}
}

/*
func TestGetHasher(t *testing.T) {
	// Test for sha256
	hasher := GetHasher("SHA256")
	if hasher == nil {
		t.Error("Expected sha256 hasher, but got nil")
	}

	// Test for unknown algorithm
	hasher = GetHasher("unknown")
	if hasher != nil {
		t.Error("Expected nil for unknown algorithm, but got non-nil")
	}
}
*/

func TestLookupNewDefaultConfiguration(t *testing.T) {
	// Test for SHA256 algorithm
	config, err := LookupDefaultConfiguration("BLAKE3")
	if err != nil {
		t.Errorf("Expected no error for BLAKE3, but got %v", err)
	}
	if config == nil || config.Algorithm != "BLAKE3" || config.Bits != 256 {
		t.Errorf("Expected BLAKE3 configuration, but got %v", config)
	}

	// Test for unknown algorithm
	config, err = LookupDefaultConfiguration("unknown")
	if err == nil {
		t.Error("Expected error for unknown algorithm, but got nil")
	}
	if config != nil {
		t.Errorf("Expected nil configuration for unknown algorithm, but got %v", config)
	}
}
