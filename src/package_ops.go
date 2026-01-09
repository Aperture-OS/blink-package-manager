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
	"io"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Aperture-OS/eyes"
)

/****************************************************/
// getPkg downloads a package recipe from the repository and saves it to the specified path
// you can use this standalone to just download recipes if you want, but usually this is
// called internally by other functions, ensuring reusing code, modularity, and less repetition
/****************************************************/

func getpkg(pkgName string, path string) error {
	eyes.Infof("Getting package recipe from local repository...")

	// acquire lock
	eyes.Infof("Acquiring lock at %s", lockPath)
	if checkLock(lockPath) {
		return fmt.Errorf("another instance is running, lock file exists at %s", lockPath)
	}

	lockErr := addLock(lockPath)
	defer removeLock(lockPath)
	if lockErr != nil {
		return fmt.Errorf("failed to create lock file at %s: %v", lockPath, lockErr)
	}

	// ensure path ends with separator
	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	repos, err := LoadRepos(configPath)
	if err != nil {
	}

	// make sure cache directories exist
	checkDirAndCreate(filepath.Join(path, "recipes"))

	// ensure repository is cloned/pulled
	if err := ensureRepo(false); err != nil {
		return fmt.Errorf("failed to update repository: %v", err)
	}

	repo, wasRepoFound := FindRepoByName(pkgName, repos)

	if !wasRepoFound {
		eyes.Errorf("Repository not found for package %s", pkgName)
	}

	destPath := filepath.Join(path, "recipes", pkgName+".json")

	// handle --force behavior: overwrite if exists
	if _, err := os.Stat(destPath); err == nil {
		eyes.Warnf("Recipe %s already exists, overwriting...", destPath)
	}

	srcPath := filepath.Join(
		repoCachePath,
		repo.Name,
		pkgName+".json",
	)

	// copy recipe from local repo cache
	input, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read package from repo cache: %v", err)
	}

	if err := os.WriteFile(destPath, input, 0644); err != nil {
		return fmt.Errorf("failed to write package to cache: %v", err)
	}

	eyes.Infof("Package %s copied to %s", pkgName, destPath)
	return nil
}

/****************************************************/
// fetchPkg fetches a package recipe from cache or repository, decodes it, and displays package info
// in addition, it returns the PackageInfo struct for further use, so you can use this function to both
// get the struct and show the info to the user, avoiding code repetition and enhancing modularity
// avoids 2 functions for fetching and displaying info separately
/****************************************************/

func fetchpkg(path string, force bool, pkgName string, quiet bool) (PackageInfo, error) {

	if !quiet {
		eyes.Infof("Fetching package %q", pkgName)
	}

	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	recipePath := filepath.Join(path, "recipes", pkgName+".json")

	if force {
		if err := os.Remove(recipePath); err == nil {
			if !quiet {
				eyes.Infof("Force flag detected, removed cached recipe at %s", recipePath)
			}
		} else if !os.IsNotExist(err) {
			if !quiet {
				eyes.Warnf("Failed to remove cached recipe.\nERR: %v", err)
			}
		}
	}

	if _, err := os.Stat(recipePath); os.IsNotExist(err) {
		if !quiet {
			eyes.Infof("Package recipe not found. Downloading...")
		}
		if err := getpkg(pkgName, path); err != nil {
			return PackageInfo{}, err
		}
	}

	f, err := os.Open(recipePath)
	if err != nil {
		eyes.Fatalf("Failed to open package recipe.\nERR: %v", err)
		return PackageInfo{}, fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	var pkg PackageInfo
	if err := json.NewDecoder(f).Decode(&pkg); err != nil {
		eyes.Fatalf("Failed to parse JSON.\nERR: %v", err)
		return PackageInfo{}, fmt.Errorf("error decoding JSON: %v", err)
	}

	if !quiet {

		repos, err := LoadRepos(configPath)
		if err != nil {
			return PackageInfo{}, fmt.Errorf("repositories could not be loaded.")
		}

		// ensure repository is cloned/pulled
		if err := ensureRepo(false); err != nil {
			return PackageInfo{}, fmt.Errorf("failed to update repository: %v", err)
		}

		repo, wasRepoFound := FindRepoByName(pkgName, repos)

		if !wasRepoFound {
			eyes.Errorf("Repository not found for package %s", pkgName)
		}

		fmt.Printf(`Repository: %q (%s)
Name:        %s
Version:     %s
Release:     %d
Description: %s
Author:      %s
License:     %s

`, repo.Name, repo.URL, pkg.Name, pkg.Version, pkg.Release, pkg.Description, pkg.Author, pkg.License)

		eyes.Infof("Package fetching completed.")
	}

	return pkg, nil
}

/****************************************************/
// install function downloads, decompresses, builds, and installs a package
// it fetches package info, downloads source, decompresses it
// it uses the getSource, decompressSource functions for modularity and to satisfy my KISS principle
// i wish golang had macros so i could avoid writing the same error handling code every single time and just have a single line for it
/****************************************************/

func install(pkgName string, force bool, path string) error {
	// manifest must exist BEFORE touching it
	if err := ensureManifest(); err != nil {
		return err
	}

	// fetch recipe
	pkg, err := fetchpkg(path, force, pkgName, false)
	if err != nil {
		return err
	}

	installed, exists, err := manifestHas(pkg.Name)
	if err != nil {
		return err
	}

	if exists && !force {
		eyes.Errorf("Package %s is already installed (version=%s release=%d). Use --force to reinstall.",
			installed.Name,
			installed.Version,
			installed.Release,
		)
		return fmt.Errorf(
			"package %s already installed (version=%s release=%d)",
			installed.Name,
			installed.Version,
			installed.Release,
		)
	}

	// mandatory deps
	if err := handleMandatoryDeps(pkg.Name, path); err != nil {
		return err
	}

	// optional deps
	if err := handleOptionalDeps(pkg.Name, path); err != nil {
		return err
	}

	packageKind := strings.ToLower(strings.TrimSpace(pkg.Build.Kind))

	switch packageKind {

	case "toCompile":

		// prepare build root
		if err := os.MkdirAll(buildRoot, 0755); err != nil {
			return err
		}

		extractRoot := filepath.Join(buildRoot, pkg.Name)

		_ = os.RemoveAll(extractRoot)
		if err := os.MkdirAll(extractRoot, 0755); err != nil {
			return err
		}

		// download source
		if err := getSource(pkg.Source.URL, force); err != nil {
			return err
		}

		srcFile := filepath.Join(sourcePath, filepath.Base(pkg.Source.URL))
		ok, err := compareSHA256(pkg.Source.Sha256, srcFile)
		if err != nil {
			return err
		}
		if !ok {
			eyes.Errorf("Source hash mismatch for %s", srcFile)
			return fmt.Errorf("source hash mismatch for %s", srcFile)
		}

		// extract
		if err := decompressSource(pkg, extractRoot); err != nil {
			return err
		}

		buildDir, err := postExtractDir(extractRoot)
		if err != nil {
			return err
		}

		if err := os.Chdir(buildDir); err != nil {
			return err
		}

		// env
		for k, v := range pkg.Build.Env {
			eyes.Infof("Setting environment variables.")
			os.Setenv(k, v)
		}

		// prepare
		for _, cmd := range pkg.Build.Prepare {
			eyes.Infof("Preparing Build environment.")
			if err := runCmd("sh", "-c", cmd); err != nil {
				return err
			}
		}

		// install
		for _, cmd := range pkg.Build.Install {
			eyes.Infof("Installing package.")
			if err := runCmd("sh", "-c", cmd); err != nil {
				return err
			}
		}

	case "preCompiled":
		eyes.Infof("Installing precompiled package %s", pkg.Name)

		// prepare build root
		if err := os.MkdirAll(buildRoot, 0755); err != nil {
			return err
		}

		extractRoot := filepath.Join(buildRoot, pkg.Name)

		_ = os.RemoveAll(extractRoot)
		if err := os.MkdirAll(extractRoot, 0755); err != nil {
			return err
		}

		// download source
		if err := getSource(pkg.Source.URL, force); err != nil {
			return err
		}

		srcFile := filepath.Join(sourcePath, filepath.Base(pkg.Source.URL))
		ok, err := compareSHA256(pkg.Source.Sha256, srcFile)
		if err != nil {
			return err
		}
		if !ok {
			eyes.Errorf("Source hash mismatch for %s", srcFile)
			return fmt.Errorf("source hash mismatch for %s", srcFile)
		}

		// extract
		if err := decompressSource(pkg, extractRoot); err != nil {
			return err
		}

		buildDir, err := postExtractDir(extractRoot)
		if err != nil {
			return err
		}

		if err := os.Chdir(buildDir); err != nil {
			return err
		}

		// extract safely
		if err := safeExtractToRoot(pkg, extractRoot); err != nil {
			return err
		}

		// default behavior: copy filesystem layout to /
		err = filepath.Walk(extractRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(extractRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join("/", rel)

		// directories
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		// skip symlinks (important for package installs)
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	})

	if err != nil {
		return err
	}

		// optional post-install commands
		for _, cmd := range pkg.Build.Install {
			if err := runCmd("sh", "-c", cmd); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unknown build kind: %s", pkg.Build.Kind)
	}

	// record install
	if err := addToManifest(pkg); err != nil {
		return err
	}
	return nil
}

/*
 *  i wonder what "finding hidden gems in blink source code" would feel like lmao
 *  well just know that this code is open source, so feel free to explore it and find any hidden gems
 *  or pull request and add a couple ;) (no mister "i wanna contribute to foss", this doesnt count as a
 *  proper contribution but if u add gems good job!) (hi from 27/12/2025 22:49 CET)
 */

/****************************************************/
// uninstall uninstalls a package
/****************************************************/

func uninstall(pkgName string, force bool, path string) error {
	// manifest must exist BEFORE touching it
	if err := ensureManifest(); err != nil {
		return err
	}

	// fetch recipe
	pkg, err := fetchpkg(path, force, pkgName, false)
	if err != nil {
		return err
	}

	_, exists, err := manifestHas(pkg.Name)
	if err != nil {
		return err
	}

	if !exists {
		eyes.Errorf("Package %s doesn't exist.", pkgName)
		return fmt.Errorf("package %s doesn't exist.", pkgName)
	}

	// prepare build root
	if err := os.MkdirAll(buildRoot, 0755); err != nil {
		return err
	}

	extractRoot := filepath.Join(buildRoot, pkg.Name)

	_ = os.RemoveAll(extractRoot)
	if err := os.MkdirAll(extractRoot, 0755); err != nil {
		return err
	}

	// download source
	if err := getSource(pkg.Source.URL, force); err != nil {
		return err
	}

	srcFile := filepath.Join(sourcePath, filepath.Base(pkg.Source.URL))
	ok, err := compareSHA256(pkg.Source.Sha256, srcFile)
	if err != nil {
		return err
	}
	if !ok {
		eyes.Errorf("Source hash mismatch for %s", srcFile)
		return fmt.Errorf("source hash mismatch for %s", srcFile)
	}

	// extract
	if err := decompressSource(pkg, extractRoot); err != nil {
		return err
	}

	buildDir, err := postExtractDir(extractRoot)
	if err != nil {
		return err
	}

	if err := os.Chdir(buildDir); err != nil {
		return err
	}

	// env
	for k, v := range pkg.Build.Env {
		eyes.Infof("Setting environment variables.")
		os.Setenv(k, v)
	}

	// install
	for _, cmd := range pkg.Build.Uninstall {
		eyes.Infof("Uninstalling package.")
		if err := runCmd("sh", "-c", cmd); err != nil {
			return err
		}
	}

	// record install
	if err := removeFromManifest(pkg); err != nil {
		return err
	}

	return nil
}

/****************************************************/
// updateAll updates all installed packages that have
// a newer Release in the repo, using the manifest's release as
// reference, it takes that and it checks if the Manifest's Release
// is smaller than the repo release, if so, install the package again
/****************************************************/
func updateAll(path string) error {

	requireRoot()

	// manifest must exist
	if err := ensureManifest(); err != nil {
		return err
	}

	// sync repositories first
	eyes.Infof("Syncing repositories...")
	if err := ensureRepo(false); err != nil {
		return fmt.Errorf("failed to sync repositories: %v", err)
	}

	m, err := loadManifest()
	if err != nil {
		return err
	}

	if len(m.Installed) == 0 {
		eyes.Infof("No installed packages found.")
		return nil
	}

	var toUpdate []InstalledPkg

	// check for updates
	for _, inst := range m.Installed {
		pkg, err := fetchpkg(path, false, inst.Name, true)
		if err != nil {
			eyes.Warnf("Failed to fetch %s, skipping: %v", inst.Name, err)
			continue
		}

		if int64(pkg.Release) > inst.Release {
			eyes.Infof(
				"Update available: %s (%d → %d)",
				inst.Name,
				inst.Release,
				pkg.Release,
			)
			toUpdate = append(toUpdate, inst)
		} else {
			eyes.Infof("Up to date: %s", inst.Name)
		}
	}

	if len(toUpdate) == 0 {
		eyes.Success("All packages are up to date.")
		return nil
	}

	eyes.Warnf("Packages to update: %d", len(toUpdate))
	for _, p := range toUpdate {
		fmt.Printf(" - %s\n", p.Name)
	}

	eyes.Warn("Proceed with update? [ (Y)es / (N)o ]: ")
	var input string
	fmt.Scanln(&input)

	switch normalizeYesNo(input) {
	case "no":
		eyes.Infof("Update aborted by user.")
		return nil
	}

	// perform updates
	for _, p := range toUpdate {
		eyes.Infof("Updating %s", p.Name)
		if err := install(p.Name, true, path); err != nil {
			return fmt.Errorf("failed to update %s: %v", p.Name, err)
		}
	}

	return nil
}
