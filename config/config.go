package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	pathname string
	Labels   map[string]map[string]interface{} `yaml:"labels"`
}

func (c *Config) Save() error {
	tmpFile, err := os.CreateTemp("", "config.*.yaml")
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	err = yaml.NewEncoder(tmpFile).Encode(c)
	if err != nil {
		return err
	}
	return os.Rename(tmpFile.Name(), c.pathname)
}

func (c *Config) Lookup(label, key string) (interface{}, bool) {
	if c.Labels == nil {
		return nil, false
	}
	if c.Labels[label] == nil {
		return nil, false
	}
	value, ok := c.Labels[label][key]
	return value, ok
}

func (c *Config) Set(category, option, value string) {
	if c.Labels == nil {
		c.Labels = make(map[string]map[string]interface{})
	}
	if c.Labels[category] == nil {
		c.Labels[category] =make(map[string]interface{})
	}
	c.Labels[category][option] = value
}

func LoadOrCreate(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
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
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}
	config.pathname = configFile
	return &config, nil
}
