package chunking

type Configuration struct {
	Algorithm  string `json:"algorithm"`   // Content-defined chunking algorithm (e.g., "rolling-hash", "fastcdc")
	MinSize    uint32 `json:"min_size"`    // Minimum chunk size
	NormalSize uint32 `json:"normal_size"` // Expected (average) chunk size
	MaxSize    uint32 `json:"max_size"`    // Maximum chunk size
}

func NewDefaultConfiguration() *Configuration {
	return &Configuration{
		Algorithm:  "FASTCDC",
		MinSize:    64 * 1024,
		NormalSize: 1 * 1024 * 1024,
		MaxSize:    4 * 1024 * 1024,
	}
}
