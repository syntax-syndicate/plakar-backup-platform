package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	pathname string
	Labels   map[string]map[string]interface{} `yaml:"labels"`
}

func LoadOrCreate(configFile string) (*Config, error) {
	f, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &Config{
				pathname: configFile,
				Labels:   make(map[string]map[string]interface{}),
			}
			return cfg, cfg.Save()
		}
		return nil, fmt.Errorf("error reading config file: %T", err)
	}
	defer f.Close()
	var config Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	config.pathname = configFile
	return &config, nil
}

func (c *Config) Save() error {
	tmpFile, err := os.CreateTemp("", "config.*.yaml")
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	err = yaml.NewEncoder(tmpFile).Encode(c)
	if err != nil {
		os.Remove(tmpFile.Name())
		return err
	}
	return os.Rename(tmpFile.Name(), c.pathname)
}

func (c *Config) Lookup(category, option string) (interface{}, bool) {
	if c.Labels == nil {
		return nil, false
	}
	if c.Labels[category] == nil {
		return nil, false
	}
	value, ok := c.Labels[category][option]
	return value, ok
}

func (c *Config) Set(category, option string, value interface{}) {
	if c.Labels == nil {
		c.Labels = make(map[string]map[string]interface{})
	}
	if c.Labels[category] == nil {
		c.Labels[category] = make(map[string]interface{})
	}
	c.Labels[category][option] = value
}
