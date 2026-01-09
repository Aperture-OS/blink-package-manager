/*
Blink, a powerful source-based package manager. Core of ApertureOS.
Want to use it for your own project?
Blink is completely FOSS (Free and Open Source),
edit, publish, use, contribute to Blink however you prefer.
Copyright (C) 2025-2026 Aperture OS

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

/****************************************************/
// LoadRepos reads a TOML file and returns a map of repository name -> RepoConfig
/****************************************************/
func LoadRepos(path string) (map[string]RepoConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", path)
	}

	var raw map[string]struct {
		GitURL string `toml:"git_url"`
		Branch string `toml:"branch"`
	}

	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode TOML: %v", err)
	}

	result := make(map[string]RepoConfig)
	for name, r := range raw {
		result[name] = RepoConfig{
			Name: name,
			URL:  r.GitURL,
			Ref:  r.Branch,
		}
	}

	return result, nil
}

/****************************************************/
// SaveRepos writes a map of RepoConfig to a TOML file
/****************************************************/
func SaveRepos(path string, repos map[string]RepoConfig) error {
	raw := make(map[string]struct {
		GitURL string `toml:"git_url"`
		Branch string `toml:"branch"`
	})

	for name, repo := range repos {
		raw[name] = struct {
			GitURL string `toml:"git_url"`
			Branch string `toml:"branch"`
		}{
			GitURL: repo.URL,
			Branch: repo.Ref,
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(raw); err != nil {
		return fmt.Errorf("failed to encode TOML: %v", err)
	}

	return nil
}
/****************************************************/
// FindRepoByName searches for a repository by name in a map of RepoConfig
/****************************************************/
func FindRepoByName(name string, repos map[string]RepoConfig) (RepoConfig, bool) {
	repo, ok := repos[name]
	return repo, ok
}

/****************************************************/
// ensureRepo makes sure all configured repositories are present and up to date
/****************************************************/
func ensureRepo(force bool) error {
	repos, err := LoadConfig() // from config.go
	if err != nil {
		return err
	}

	for name, repo := range repos {
		repoPath := filepath.Join(repoCachePath, name)

		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			// clone
			if err := cloneRepo(repo.URL, repo.Ref, repoPath); err != nil {
				return err
			}
			continue
		}

		if force {
			if err := resetRepo(repoPath, repo.Ref); err != nil {
				return err
			}
			continue
		}

		// pull
		if err := pullRepo(repoPath); err != nil {
			return err
		}
	}

	return nil
}

/****************************************************/
// some commands boilerplate functions to improve KISS 
// and readability
/****************************************************/
func cloneRepo(url, ref, dest string) error {
	cmd := exec.Command("git", "clone", "-b", ref, url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func pullRepo(path string) error {
	cmd := exec.Command("git", "-C", path, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resetRepo(path, ref string) error {
	cmd := exec.Command("git", "-C", path, "fetch", "--all")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "-C", path, "reset", "--hard", "origin/"+ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
