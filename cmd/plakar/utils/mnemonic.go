package utils

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	_ "embed"
)

//go:embed mnemonic.txt
var wordlistTxt string
var wordlist []string
var wordToIndex map[string]int

func init() {
	// Trim possible trailing newline
	wordlist = strings.Split(strings.TrimSpace(wordlistTxt), "\n")
	if len(wordlist) != 2048 {
		panic(fmt.Sprintf("mnemonic.txt should have 2048 entries, got %d", len(wordlist)))
	}
	wordToIndex = make(map[string]int, len(wordlist))
	for i, w := range wordlist {
		wordToIndex[w] = i
	}
}

// GenerateMnemonic converts a secret (entropy) buffer into a BIP-39 style mnemonic.
// secret must be a multiple of 4 bytes (32 bits); for example, 16, 20, 24, 28, or 32 bytes.
func GenerateMnemonic(secret []byte) (string, error) {
	ent := len(secret) * 8
	// BIP-39 requires entropy between 128 and 256 bits, multiple of 32
	if ent%32 != 0 || ent < 128 || ent > 256 {
		return "", errors.New("entropy must be 128-256 bits and a multiple of 32")
	}
	// Compute checksum length
	cs := ent / 32

	// Compute SHA-256 checksum
	hash := sha256.Sum256(secret)
	// Build bit buffer: secret bits + checksum bits
	bitLen := ent + cs
	// Buffer of whole bytes: ceil(bitLen/8)
	buf := make([]byte, (bitLen+7)/8)
	copy(buf, secret)
	// Append checksum bits at the end of buffer
	// For full bytes, we OR the top cs bits of hash[0]
	buf[len(secret)] = hash[0] & (0xFF << (8 - cs))

	// Split into words of 11 bits
	nWords := bitLen / 11
	mnemonic := make([]string, nWords)
	for i := 0; i < nWords; i++ {
		bitIndex := i * 11
		// Determine which bytes to read
		bytePos := bitIndex / 8
		bitOffset := bitIndex % 8

		// Read enough bits into a 32-bit integer
		// we need up to 11 bits, so read three bytes to be safe
		var w uint32
		w = uint32(buf[bytePos]) << 16
		if bytePos+1 < len(buf) {
			w |= uint32(buf[bytePos+1]) << 8
		}
		if bytePos+2 < len(buf) {
			w |= uint32(buf[bytePos+2])
		}
		// Shift to align the 11-bit value at MSB, then drop extra bits
		shift := 24 - 11 - bitOffset
		index := (w >> shift) & 0x7FF

		mnemonic[i] = wordlist[index]
	}

	return strings.Join(mnemonic, " "), nil
}

func RecoverSecret(mnemonic string) ([]byte, error) {
	words := strings.Fields(mnemonic)
	n := len(words)
	if n == 0 || n%3 != 0 {
		return nil, errors.New("invalid mnemonic word count")
	}
	// Calculate ent and cs from word count: (ent+cs)/11 = n, and cs = ent/32
	bitLen := n * 11
	// ent = bitLen * 32 / 33
	ent := bitLen * 32 / 33
	cs := ent / 32

	// Build full bit buffer
	buf := make([]bool, bitLen)
	for i, w := range words {
		idx, ok := wordToIndex[w]
		if !ok {
			return nil, fmt.Errorf("invalid mnemonic word: %s", w)
		}
		// Fill 11 bits
		for b := 0; b < 11; b++ {
			if idx&(1<<(10-b)) != 0 {
				buf[i*11+b] = true
			}
		}
	}

	// Extract entropy bits
	entropy := make([]byte, ent/8)
	for i := 0; i < ent; i++ {
		if buf[i] {
			entropy[i/8] |= 1 << (7 - (i % 8))
		}
	}

	// Optional: verify checksum
	hash := sha256.Sum256(entropy)
	for i := 0; i < cs; i++ {
		bit := (hash[0] >> (7 - i)) & 1
		if buf[ent+i] != (bit == 1) {
			return nil, errors.New("checksum mismatch")
		}
	}

	return entropy, nil
}

func ASCIIGrid(hash []byte, width, height int) (string, error) {
	bits := bytesToBits(hash) // e.g. []bool
	if len(bits) > width*height {
		return "", fmt.Errorf("grid too small")
	}
	var out strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if bits[i] {
				out.WriteRune('█')
			} else {
				out.WriteRune(' ')
			}
		}
		out.WriteRune('\n')
	}
	return out.String(), nil
}

func RecoverHashFromGrid(grid string) ([]byte, error) {
	lines := strings.Split(strings.TrimRight(grid, "\n"), "\n")
	width := len(lines[0])
	var bits []bool
	for _, line := range lines {
		if len(line) != width {
			return nil, errors.New("inconsistent row length")
		}
		for _, r := range line {
			bits = append(bits, r == '█')
		}
	}
	return bitsToBytes(bits), nil
}

// bitsToBytes packs a slice of booleans (big-endian per byte) into a []byte.
func bitsToBytes(bits []bool) []byte {
	n := (len(bits) + 7) / 8
	out := make([]byte, n)
	for i, bit := range bits {
		if bit {
			byteIndex := i / 8
			bitPos := 7 - (i % 8) // highest bit first
			out[byteIndex] |= 1 << bitPos
		}
	}
	return out
}

// bytesToBits unpacks a []byte into a []bool, big-endian per byte.
func bytesToBits(data []byte) []bool {
	bits := make([]bool, len(data)*8)
	for i, b := range data {
		for j := 0; j < 8; j++ {
			// check bit (7-j)
			bits[i*8+j] = (b & (1 << (7 - j))) != 0
		}
	}
	return bits
}
