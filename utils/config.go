package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PlakarKorp/kloset/config"
)

type configHandler struct {
	Path   string
	Config *config.Config
}

func newConfigHandler(path string) *configHandler {
	return &configHandler{
		Path: path,
	}
}

func (cl *configHandler) Load() (*config.Config, error) {
	cl.Config = config.NewConfig()
	err := cl.load("sources.json", &cl.Config.Sources)
	if err != nil {
		return nil, err
	}
	err = cl.load("destinations.json", &cl.Config.Destinations)
	if err != nil {
		return nil, err
	}
	err = cl.load("klosets.json", &cl.Config.Repositories)
	if err != nil {
		return nil, err
	}
	for k, v := range cl.Config.Repositories {
		if _, ok := v[".default"]; ok {
			cl.Config.DefaultRepository = k
			delete(v, ".default")
		}
	}

	return cl.Config, nil
}

func (cl *configHandler) Save(cfg *config.Config) error {
	err := cl.save("sources.json", cfg.Sources)
	if err != nil {
		return err
	}
	err = cl.save("destinations.json", cfg.Destinations)
	if err != nil {
		return err
	}
	for k, v := range cfg.Repositories {
		if k == cfg.DefaultRepository {
			v[".default"] = "yes"
		}
	}
	err = cl.save("klosets.json", cfg.Repositories)
	if err != nil {
		return err
	}
	return nil
}

func (cl *configHandler) load(filename string, dst any) error {
	path := filepath.Join(cl.Path, filename)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error reading config file: %w", err)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(dst)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

func (cl *configHandler) save(filename string, src any) error {
	path := filepath.Join(cl.Path, filename)
	tmpFile, err := os.CreateTemp(cl.Path, "config.*.json")
	if err != nil {
		return err
	}

	err = json.NewEncoder(tmpFile).Encode(src)
	tmpFile.Close()

	if err == nil {
		err = os.Rename(tmpFile.Name(), path)
	}

	if err != nil {
		os.Remove(tmpFile.Name())
		return err
	}

	return nil
}

func LoadConfig(configDir string) (*config.Config, error) {
	cl := newConfigHandler(configDir)
	cfg, err := cl.Load()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveConfig(configDir string, cfg *config.Config) error {
	return newConfigHandler(configDir).Save(cfg)
}
