package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/PlakarKorp/kloset/config"
)

type OldConfig struct {
	DefaultRepository string                             `yaml:"default-repo"`
	Repositories      map[string]config.RepositoryConfig `yaml:"repositories"`
	Remotes           map[string]config.SourceConfig     `yaml:"remotes"`
}

func LoadOldConfigIfExists(configFile string) (*config.Config, error) {
	f, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading old config file: %w", err)
	}
	defer f.Close()

	var old OldConfig
	if err := yaml.NewDecoder(f).Decode(&old); err != nil {
		return nil, fmt.Errorf("failed to parse old config file: %w", err)
	}

	cfg := config.NewConfig()
	cfg.Repositories = old.Repositories
	cfg.Sources = old.Remotes
	return cfg, nil
}
