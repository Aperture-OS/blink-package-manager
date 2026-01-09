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

/****************************************************/
// Manifest creation and dependency handling functions
// a manifest is a TOML file that keeps track of installed packages,
// their versions, and installation timestamps
// this is useful for managing installed packages, checking for updates, and handling dependencies
/****************************************************/
package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/Aperture-OS/eyes"
)

// ensureManifest makes sure the manifest file exists and creates it if it doesn't
func ensureManifest() error {
	eyes.Infof("Ensuring manifest exists at %s", manifestPath)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		m := Manifest{Installed: []InstalledPkg{}}
		file, err := os.Create(manifestPath)
		if err != nil {
			return err
		}
		defer file.Close()
		return toml.NewEncoder(file).Encode(m)
	}

	return nil
}

/****************************************************/
// loadManifest loads the manifest from disk
/****************************************************/
func loadManifest() (Manifest, error) {
	eyes.Infof("Loading manifest")

	var m Manifest
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return Manifest{Installed: []InstalledPkg{}}, nil
	}

	if _, err := toml.DecodeFile(manifestPath, &m); err != nil {
		return m, err
	}

	return m, nil
}

/****************************************************/
// saveManifest writes the manifest back to disk safely
/****************************************************/
func saveManifest(m Manifest) error {
	eyes.Infof("Saving manifest (%d packages)", len(m.Installed))

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}

	tmp := manifestPath + ".tmp"

	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(m); err != nil {
		return err
	}

	return os.Rename(tmp, manifestPath)
}

// manifestHas checks if a package is already in the manifest
func manifestHas(name string) (*InstalledPkg, bool, error) {
	m, err := loadManifest()
	if err != nil {
		return nil, false, err
	}

	for _, p := range m.Installed {
		if p.Name == name {
			return &p, true, nil
		}
	}

	return nil, false, nil
}

/****************************************************/
// isInstalled checks if a package is installed by name
/****************************************************/
func isInstalled(pkg string) bool {
	_, ok, err := manifestHas(pkg)
	return err == nil && ok
}

/****************************************************/
// addToManifest adds a package to the manifest if it doesn't already exist
/****************************************************/
func addToManifest(pkg PackageInfo) error {
	eyes.Infof("adding %s to manifest", pkg.Name)

	m, err := loadManifest()
	if err != nil {
		return err
	}

	for _, p := range m.Installed {
		if p.Name == pkg.Name {
			eyes.Warnf("%s already recorded in manifest", pkg.Name)
			return nil
		}
	}

	m.Installed = append(m.Installed, InstalledPkg{
		Name:    pkg.Name,
		Version: pkg.Version,
		Release: int64(pkg.Release),
	})

	return saveManifest(m)
}

/****************************************************/
// removeFromManifest removes a package from the manifest if it exists
/****************************************************/
func removeFromManifest(pkg PackageInfo) error {
	eyes.Infof("removing %s from manifest", pkg.Name)

	m, err := loadManifest()
	if err != nil {
		return err
	}

	found := false
	newInstalled := make([]InstalledPkg, 0, len(m.Installed))

	for _, p := range m.Installed {
		if p.Name == pkg.Name {
			found = true
			continue // skip the package we want to remove
		}
		newInstalled = append(newInstalled, p)
	}

	if !found {
		eyes.Warnf("%s not found in manifest", pkg.Name)
		return nil
	}

	m.Installed = newInstalled
	return saveManifest(m)
}
