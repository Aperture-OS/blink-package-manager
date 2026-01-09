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
	"strconv"
	"strings"

	"github.com/Aperture-OS/eyes"
	"github.com/Aperture-OS/togosort-dfs"
)

// note to others: this was a pain in the ass to implement but it works (hopefully)
// have fun maintaining if youre a maintainer :D 
// TIP: Check out github.com/Aperture-OS/togosort-dfs docs and comments in source code


/****************************************************/
//
// Recursive helper to build dependency graph
// IMPORTANT: AddEdge(A, B) == A depends on B
//
/****************************************************/
func buildDepGraph(
	graph *togosort.Graph,
	pkgName string,
	path string,
	visited map[string]bool,
) error {
	if visited[pkgName] {
		return nil
	}
	visited[pkgName] = true

	pkg, err := fetchpkg(path, false, pkgName, true)
	if err != nil {
		return fmt.Errorf("failed to fetch package %s: %v", pkgName, err)
	}

	for dep := range pkg.Dependencies {
		// pkgName depends on dep
		graph.AddEdge(pkgName, dep)

		if err := buildDepGraph(graph, dep, path, visited); err != nil {
			return err
		}
	}

	return nil
}

/****************************************************/
//
// Handle mandatory dependencies (DFS + topo)
//
/****************************************************/
func handleMandatoryDeps(pkgName, path string) error {
	graph := togosort.NewGraph()
	visited := make(map[string]bool)

	if err := buildDepGraph(graph, pkgName, path, visited); err != nil {
		return err
	}

	// cycle detection
	if err := graph.DFS([]string{pkgName}); err != nil {
		return fmt.Errorf("dependency cycle detected: %v", err)
	}

	order := graph.TopoSort()

	var missing []string
	for _, dep := range order {
		if dep == pkgName {
			continue
		}
		if !isInstalled(dep) {
			missing = append(missing, dep)
		} else {
			eyes.Infof("Dependency %s already installed", dep)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	eyes.Warnf("Missing mandatory dependencies: %v", missing)
	eyes.Warnf("Mandatory dependencies are required for proper functionality.")
	eyes.Warn("Do you want to install mandatory dependencies? [ (Y)es / (N)o ]: ")

	var input string
	fmt.Scanln(&input)
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "n", "no":
		eyes.Fatalf("Cannot continue without mandatory dependencies.")
	case "bypass-donotuse":
		eyes.Warnf(`[DEVELOPER ONLY/INSECURE] Bypassing mandatory dependencies check (press CTRL+C to cancel).
This is not secure, your package could break! To fix this properly rerun the command you just ran and install the missing dependencies
By entering 'y' when prompted to install mandatory dependencies. If you are having issues afterwards please contact support in
our Discord server. Run 'blink support' for more informations.`)
		return nil
	}

	for _, dep := range order {
		if dep == pkgName || isInstalled(dep) {
			continue
		}
		eyes.Infof("Installing dependency %s", dep)
		if err := install(dep, false, path); err != nil {
			return fmt.Errorf("failed to install dependency %s: %v", dep, err)
		}
	}

	return nil
}

/****************************************************/
//
// Handle optional dependencies (DFS + topo per choice)
//
/****************************************************/
func handleOptionalDeps(pkgName string, path string) error {
	pkg, err := fetchpkg(path, false, pkgName, true)
	if err != nil {
		return fmt.Errorf("failed to fetch package %s: %v", pkgName, err)
	}

	for _, group := range pkg.OptDeps {
		var installed []string
		var notInstalled []string

		for _, opt := range group.Options {
			if isInstalled(opt) {
				installed = append(installed, opt)
			} else {
				notInstalled = append(notInstalled, opt)
			}
		}

		eyes.Infof("Optional dependency group %d: %s", group.ID, group.Description)

		if len(installed) > 0 {
			eyes.Infof("Already installed: %v", installed)
		}

		defaultChoice := "1"
		if len(installed) > 0 || len(notInstalled) == 0 {
			defaultChoice = "0"
		}

		fmt.Println("[ 0 ] None")
		for i, opt := range notInstalled {
			fmt.Printf("[ %d ] %s\n", i+1, opt)
		}

		eyes.Warnf("Select optional dependency (default=%s): ", defaultChoice)

		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input == "" {
			input = defaultChoice
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 0 || choice > len(notInstalled) {
			eyes.Warnf("Invalid choice, skipping optional group %d", group.ID)
			continue
		}

		if choice == 0 {
			eyes.Infof("Skipping optional group %d", group.ID)
			continue
		}

		selected := notInstalled[choice-1]

		graph := togosort.NewGraph()
		visited := make(map[string]bool)

		if err := buildDepGraph(graph, selected, path, visited); err != nil {
			return err
		}

		if err := graph.DFS([]string{selected}); err != nil {
			return fmt.Errorf("dependency cycle detected: %v", err)
		}

		order := graph.TopoSort()

		for _, dep := range order {
			if isInstalled(dep) {
				continue
			}
			eyes.Infof("Installing optional dependency %s", dep)
			if err := install(dep, false, path); err != nil {
				return fmt.Errorf("failed to install optional dependency %s: %v", dep, err)
			}
		}
	}

	return nil
}
