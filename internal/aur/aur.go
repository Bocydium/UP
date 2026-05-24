package aur

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aapollo/up/internal/cache"
	"github.com/aapollo/up/internal/cli"
	"github.com/aapollo/up/internal/pacman"
	"github.com/aapollo/up/internal/ui"
)

const aurRPC = "https://aur.archlinux.org/rpc/v5"

// Package represents an AUR package from the RPC API.
type Package struct {
	Name        string `json:"Name"`
	Version     string `json:"Version"`
	Description string `json:"Description"`
	URL         string `json:"URL"`
	Maintainer  string `json:"Maintainer"`
	Votes       int    `json:"NumVotes"`
	OutOfDate   int64  `json:"OutOfDate"`
	Depends     []string
}

// Search finds the best matching AUR package.
func Search(pkg string) (*Package, error) {
	results, err := SearchMulti(pkg)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	for _, r := range results {
		if strings.EqualFold(r.Name, pkg) {
			return &r, nil
		}
	}
	return &results[0], nil
}

// SearchMulti searches AUR and returns multiple results.
func SearchMulti(query string) ([]Package, error) {
	url := fmt.Sprintf("%s/search/%s?by=name-desc", aurRPC, query)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []Package `json:"results"`
		Type    string    `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Type == "error" {
		return nil, fmt.Errorf("AUR search error")
	}
	return result.Results, nil
}

// Info gets detailed info about a specific AUR package.
func Info(pkg string) (*Package, error) {
	url := fmt.Sprintf("%s/info/%s", aurRPC, pkg)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []Package `json:"results"`
		Type    string    `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("package not found")
	}
	return &result.Results[0], nil
}

// Download clones the AUR git repo for a package.
func Download(pkg *Package) (string, error) {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "up", "aur")
	os.MkdirAll(cacheDir, 0755)

	buildDir := filepath.Join(cacheDir, pkg.Name)
	os.RemoveAll(buildDir)

	cmd := exec.Command("git", "clone",
		fmt.Sprintf("https://aur.archlinux.org/%s.git", pkg.Name),
		buildDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buildDir, nil
}

// Build runs makepkg in the build directory.
func Build(buildDir string, flags cli.Flags) error {
	args := []string{"-s", "--noconfirm"}
	if flags.NoCheck {
		args = append(args, "--nocheck")
	}

	cmd := exec.Command("makepkg", args...)
	cmd.Dir = buildDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// InstallPkg installs the built package with pacman.
func InstallPkg(buildDir string, flags cli.Flags) error {
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return err
	}

	var pkgFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pkg.tar.zst") {
			pkgFile = filepath.Join(buildDir, entry.Name())
			break
		}
	}
	if pkgFile == "" {
		return fmt.Errorf("no package file found in %s", buildDir)
	}

	args := []string{"-U", "--needed"}
	if flags.NoConfirm {
		args = append(args, "--noconfirm")
	}
	args = append(args, pkgFile)

	cmd := exec.Command("sudo", append([]string{"pacman"}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// InstallWithCache installs an AUR package, using binary cache if available.
func InstallWithCache(pkg *Package, flags cli.Flags) error {
	// Download PKGBUILD
	ui.Step("Downloading %s...", pkg.Name)
	buildDir, err := Download(pkg)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Compute build hash
	buildHash, err := cache.HashBuildDir(buildDir)
	if err != nil {
		ui.Step("Could not hash build dir, building from scratch")
		buildHash = ""
	}

	// Check binary cache
	if buildHash != "" {
		if cachedPath, found := cache.FindCachedBinary(pkg.Name, buildHash); found {
			ui.Success("Found cached binary for %s", pkg.Name)
			args := []string{"-U", "--needed"}
			if flags.NoConfirm {
				args = append(args, "--noconfirm")
			}
			args = append(args, cachedPath)
			cmd := exec.Command("sudo", append([]string{"pacman"}, args...)...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
	}

	// Build
	ui.Step("Building %s...", pkg.Name)
	if err := Build(buildDir, flags); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Find built package
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return err
	}
	var pkgFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pkg.tar.zst") {
			pkgFile = filepath.Join(buildDir, entry.Name())
			break
		}
	}
	if pkgFile == "" {
		return fmt.Errorf("no package file found after build")
	}

	// Cache the binary
	if buildHash != "" {
		if err := cache.CacheBinary(pkgFile, pkg.Name, buildHash); err == nil {
			ui.Step("Cached binary for future installs")
		}
	}

	// Install
	ui.Step("Installing %s...", pkg.Name)
	args := []string{"-U", "--needed"}
	if flags.NoConfirm {
		args = append(args, "--noconfirm")
	}
	args = append(args, pkgFile)
	cmd := exec.Command("sudo", append([]string{"pacman"}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// UpdateAll checks for AUR updates and rebuilds outdated packages.
func UpdateAll() error {
	pkgs, err := pacman.GetInstalledAURPackages()
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		ui.Info("No AUR packages installed")
		return nil
	}

	ui.Step("Checking %d AUR packages...", len(pkgs))

	var toUpdate []struct {
		name   string
		oldVer string
		newVer string
	}

	for _, pkg := range pkgs {
		info, err := Info(pkg)
		if err != nil {
			continue
		}
		cmd := exec.Command("pacman", "-Q", pkg)
		out, _ := cmd.Output()
		installed := strings.TrimSpace(string(out))
		parts := strings.Fields(installed)
		if len(parts) >= 2 {
			if parts[1] != info.Version {
				toUpdate = append(toUpdate, struct {
					name   string
					oldVer string
					newVer string
				}{name: pkg, oldVer: parts[1], newVer: info.Version})
			}
		}
	}

	if len(toUpdate) == 0 {
		ui.Success("All AUR packages up to date")
		return nil
	}

	ui.Header("AUR updates available (%d)", len(toUpdate))
	for _, u := range toUpdate {
		fmt.Printf("  %s%s%s %s → %s\n", ui.Yellow(), u.name, ui.Reset(), u.oldVer, u.newVer)
	}

	if !ui.Prompt("Update AUR packages?") {
		return nil
	}

	cached := 0
	built := 0
	for _, u := range toUpdate {
		info, err := Info(u.name)
		if err != nil {
			ui.Error("Failed to get info for %s: %v", u.name, err)
			continue
		}
		if err := InstallWithCache(info, cli.Flags{}); err != nil {
			ui.Error("Failed to update %s: %v", u.name, err)
			continue
		}
		// Track cache vs build
		buildHash, _ := cache.HashBuildDir(filepath.Join(os.Getenv("HOME"), ".cache", "up", "aur", u.name))
		if _, found := cache.FindCachedBinary(u.name, buildHash); found {
			cached++
		} else {
			built++
		}
	}

	ui.Stats(cached, built, len(toUpdate))
	return nil
}
