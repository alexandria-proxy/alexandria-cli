package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Lang string `json:"lang"`
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "alexandria"), nil
}

func Load() (Config, error) {
	var c Config
	d, err := dir()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(filepath.Join(d, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	err = json.Unmarshal(data, &c)
	return c, err
}

func Save(c Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, "config.json"), data, 0600)
}
