package main

import (
	"os"
	"path/filepath"

	"github.com/Aperture-OS/eyes"
	"github.com/BurntSushi/toml"
)

/****************************************************/
// CreateDefaultConfig writes the default repository config to configPath
/****************************************************/
func CreateDefaultConfig() error {
	if configPath == "" {
		return os.ErrInvalid
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create the config file
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Wrap defaultRepoConfig from globals.go
	if err := toml.NewEncoder(file).Encode(defaultRepoConfig); err != nil {
		return err
	}

	eyes.Infof("Default repository config created at %s", configPath)
	return nil
}

/****************************************************/
// LoadConfig loads the repository config from configPath
/****************************************************/
func LoadConfig() (map[string]RepoConfig, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		eyes.Infof("Config file not found. Creating default config at %s", configPath)
		if err := CreateDefaultConfig(); err != nil {
			return nil, err
		}
	}

	var repos map[string]RepoConfig
	if _, err := toml.DecodeFile(configPath, &repos); err != nil {
		return nil, err
	}

	if len(repos) == 0 {
		return nil, os.ErrInvalid
	}

	eyes.Infof("Loaded %d repositories from %s", len(repos), configPath)
	return repos, nil
}
