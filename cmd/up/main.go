package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aapollo/up/internal/aur"
	"github.com/aapollo/up/internal/cache"
	"github.com/aapollo/up/internal/cli"
	"github.com/aapollo/up/internal/flatpak"
	"github.com/aapollo/up/internal/pacman"
	"github.com/aapollo/up/internal/security"
	"github.com/aapollo/up/internal/ui"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "inst":
		handleInstall(args)
	case "remo":
		handleRemove(args)
	case "upda":
		handleUpdate()
	case "search", "se":
		handleSearch(args)
	case "info":
		handleInfo(args)
	case "cache":
		handleCache(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "up: unknown command %q\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`up - the fast, secure package manager

Commands:
  up inst <package>       Install a package (official repos + AUR)
  up remo <package>       Remove a package cleanly
  up upda                 Update all packages (pacman + AUR + flatpak)
  up search <query>       Search for packages
  up info <package>       Show package details
  up cache                Show cache size
  up cache clean          Clean old cached builds

Flags:
  --noconfirm             Skip confirmation prompts
  --nocheck               Skip security checks (not recommended)
  --needed                Only install if not already installed`)
}

func handleInstall(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "up: inst requires a package name")
		os.Exit(1)
	}

	pkg := args[0]
	flags := cli.ParseFlags(args[1:])

	ui.Header("Installing %s", pkg)

	if flags.Needed && pacman.IsInstalled(pkg) {
		ui.Info("%s is already installed, skipping", pkg)
		return
	}

	if pacman.InOfficialRepo(pkg) {
		ui.Step("Found in official repositories")
		if err := pacman.Install(pkg, flags); err != nil {
			ui.Fatal("Installation failed: %v", err)
		}
		ui.Success("Installed %s from official repos", pkg)
		return
	}

	ui.Step("Searching AUR...")
	result, err := aur.Search(pkg)
	if err != nil {
		ui.Fatal("AUR search failed: %v", err)
	}

	if result == nil {
		ui.Fatal("Package %s not found in official repos or AUR", pkg)
	}

	ui.Step("Found in AUR: %s (%s)", result.Name, result.Version)

	if !flags.NoCheck {
		ui.Step("Running security checks...")
		if err := security.ScanAURPackage(result); err != nil {
			ui.Fatal("Security check failed: %v", err)
		}
		ui.Success("Security checks passed")
	}

	if err := aur.InstallWithCache(result, flags); err != nil {
		ui.Fatal("Install failed: %v", err)
	}

	ui.Success("Installed %s", pkg)
}

func handleRemove(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "up: remo requires a package name")
		os.Exit(1)
	}

	pkg := args[0]
	flags := cli.ParseFlags(args[1:])

	ui.Header("Removing %s", pkg)

	if !pacman.IsInstalled(pkg) {
		ui.Fatal("Package %s is not installed", pkg)
	}

	ui.Step("Removing package and unused dependencies...")
	if err := pacman.Remove(pkg, flags); err != nil {
		ui.Fatal("Removal failed: %v", err)
	}

	ui.Success("Removed %s", pkg)
}

func handleUpdate() {
	ui.Header("Updating all packages")

	ui.Step("Updating official repositories...")
	if err := pacman.Update(); err != nil {
		ui.Error("Pacman update failed: %v", err)
	}

	ui.Step("Checking AUR updates...")
	if err := aur.UpdateAll(); err != nil {
		ui.Error("AUR update failed: %v", err)
	}

	ui.Step("Updating flatpak packages...")
	if err := flatpak.Update(); err != nil {
		ui.Error("Flatpak update failed: %v", err)
	}

	ui.Success("Update complete")
}

func handleSearch(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "up: search requires a query")
		os.Exit(1)
	}

	query := strings.Join(args, " ")
	ui.Header("Searching for %q", query)

	ui.Step("Official repositories:")
	if results, err := pacman.Search(query); err == nil {
		for _, r := range results {
			fmt.Printf("  %s/%s %s\n    %s\n", r.Repo, r.Name, r.Version, r.Description)
		}
	}

	ui.Step("AUR:")
	if results, err := aur.SearchMulti(query); err == nil {
		for _, r := range results {
			fmt.Printf("  aur/%s %s (+%d)\n    %s\n", r.Name, r.Version, r.Votes, r.Description)
		}
	}
}

func handleInfo(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "up: info requires a package name")
		os.Exit(1)
	}

	pkg := args[0]

	if info, err := pacman.Info(pkg); err == nil {
		ui.Header("%s/%s %s", info.Repo, info.Name, info.Version)
		fmt.Printf("  Description: %s\n", info.Description)
		fmt.Printf("  URL: %s\n", info.URL)
		fmt.Printf("  License: %s\n", info.License)
		fmt.Printf("  Depends: %s\n", strings.Join(info.Depends, ", "))
		return
	}

	if info, err := aur.Info(pkg); err == nil {
		ui.Header("aur/%s %s", info.Name, info.Version)
		fmt.Printf("  Description: %s\n", info.Description)
		fmt.Printf("  URL: %s\n", info.URL)
		fmt.Printf("  Votes: %d\n", info.Votes)
		fmt.Printf("  Maintainer: %s\n", info.Maintainer)
		fmt.Printf("  Depends: %s\n", strings.Join(info.Depends, ", "))
		return
	}

	ui.Fatal("Package %s not found", pkg)
}

func handleCache(args []string) {
	if len(args) == 0 {
		size, err := cache.Size()
		if err != nil {
			ui.Fatal("Failed to get cache size: %v", err)
		}
		ui.Header("Cache size: %.2f MB", float64(size)/(1024*1024))
		return
	}

	if args[0] == "clean" {
		ui.Header("Cleaning cache")
		if err := cache.Clean(10); err != nil {
			ui.Fatal("Failed to clean cache: %v", err)
		}
		ui.Success("Cache cleaned (kept last 10 builds)")
		return
	}

	ui.Fatal("Unknown cache command: %s", args[0])
}
