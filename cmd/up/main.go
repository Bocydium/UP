package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aapollo/up/internal/aur"
	"github.com/aapollo/up/internal/backup"
	"github.com/aapollo/up/internal/cache"
	"github.com/aapollo/up/internal/cli"
	"github.com/aapollo/up/internal/db"
	"github.com/aapollo/up/internal/diff"
	"github.com/aapollo/up/internal/extract"
	"github.com/aapollo/up/internal/flatpak"
	"github.com/aapollo/up/internal/health"
	"github.com/aapollo/up/internal/mirror"
	"github.com/aapollo/up/internal/pacman"
	"github.com/aapollo/up/internal/plan"
	"github.com/aapollo/up/internal/security"
	"github.com/aapollo/up/internal/tree"
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
	case "diff":
		handleDiff(args)
	case "tree":
		handleTree(args)
	case "backup":
		handleBackup(args)
	case "plan":
		handlePlan(args)
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
  up diff <package>       Show file changes before update
  up tree <package>       Show dependency tree
  up backup               Save package list snapshot
  up plan <package>       Dry-run: show what would happen
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

	// Check for --plan flag
	if flags.Plan {
		p, err := plan.InstallPlan(pkg)
		if err != nil {
			ui.Fatal("Plan failed: %v", err)
		}
		p.Print()
		return
	}

	ui.Header("Installing %s", pkg)

	// Open local DB
	database, err := db.Open()
	if err != nil {
		ui.Fatal("Failed to open database: %v", err)
	}

	if flags.Needed && database.IsInstalled(pkg) {
		ui.Info("%s is already installed, skipping", pkg)
		return
	}

	// Try official repos first (native, no pacman subprocess)
	ui.Step("Checking official repositories...")
	mpkg, err := mirror.FindPackage(pkg)
	if err == nil && mpkg != nil {
		ui.Step("Found %s/%s %s", mpkg.Repo, mpkg.Name, mpkg.Version)

		// Download
		cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "up", "pkgs")
		os.MkdirAll(cacheDir, 0755)
		pkgPath := filepath.Join(cacheDir, mpkg.Filename)

		if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
			if err := mirror.Download(mpkg, pkgPath); err != nil {
				ui.Fatal("Download failed: %v", err)
			}
		} else {
			ui.Success("Using cached package")
		}

		// Extract (native, no pacman -U)
		ui.Step("Extracting...")
		files, err := extract.ListFiles(pkgPath)
		if err != nil {
			ui.Fatal("Failed to read package: %v", err)
		}

		if err := extract.Install(pkgPath, "/"); err != nil {
			ui.Fatal("Install failed: %v", err)
		}

		// Record in DB
		database.Install(mpkg.Name, mpkg.Version, mpkg.Repo, files)
		database.Save()

		ui.Success("Installed %s", pkg)
		return
	}

	// Fall back to AUR
	ui.Step("Searching AUR...")
	result, err := aur.Search(pkg)
	if err != nil {
		ui.Fatal("AUR search failed: %v", err)
	}

	if result == nil {
		ui.Fatal("Package %s not found in official repos or AUR", pkg)
	}

	ui.Step("Found in AUR: %s (%s)", result.Name, result.Version)

	// Show health score
	score := health.Calculate(result)
	fmt.Printf("  Health: %s%s%s %d/100 %s\n",
		score.Color(), score.Bar(), ui.Reset(),
		score.Value, score.Verdict)

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

	// Check local DB first
	database, err := db.Open()
	if err != nil {
		ui.Fatal("Failed to open database: %v", err)
	}

	record, ok := database.Get(pkg)
	if !ok {
		// Fall back to pacman check
		if !pacman.IsInstalled(pkg) {
			ui.Fatal("Package %s is not installed", pkg)
		}
		ui.Step("Removing via pacman...")
		if err := pacman.Remove(pkg, flags); err != nil {
			ui.Fatal("Removal failed: %v", err)
		}
		ui.Success("Removed %s", pkg)
		return
	}

	// Native removal: delete tracked files
	ui.Step("Removing %d tracked files...", len(record.Files))
	removed := 0
	for _, file := range record.Files {
		path := filepath.Join("/", file)
		if err := os.Remove(path); err == nil {
			removed++
		}
	}

	// Clean empty directories
	ui.Step("Cleaning empty directories...")

	// Remove from DB
	database.Remove(pkg)
	database.Save()

	ui.Success("Removed %s (%d files)", pkg, removed)
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
			score := health.Calculate(&r)
			fmt.Printf("  aur/%s %s (+%d) %s%d%s/100\n    %s\n",
				r.Name, r.Version, r.Votes,
				score.Color(), score.Value, ui.Reset(),
				r.Description)
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
		score := health.Calculate(info)
		fmt.Printf("  Health:     %s%s%s %d/100 %s\n",
			score.Color(), score.Bar(), ui.Reset(),
			score.Value, score.Verdict)
		fmt.Printf("  Description: %s\n", info.Description)
		fmt.Printf("  URL: %s\n", info.URL)
		fmt.Printf("  Votes: %d\n", info.Votes)
		fmt.Printf("  Maintainer: %s\n", info.Maintainer)
		fmt.Printf("  Depends: %s\n", strings.Join(info.Depends, ", "))
		return
	}

	ui.Fatal("Package %s not found", pkg)
}

func handleDiff(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "up: diff requires a package name")
		os.Exit(1)
	}
	if err := diff.Show(args[0]); err != nil {
		ui.Fatal("%v", err)
	}
}

func handleTree(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "up: tree requires a package name")
		os.Exit(1)
	}
	if err := tree.Show(args[0]); err != nil {
		ui.Fatal("%v", err)
	}
}

func handleBackup(args []string) {
	if len(args) == 0 {
		if err := backup.List(); err != nil {
			ui.Fatal("%v", err)
		}
		return
	}

	switch args[0] {
	case "create", "save":
		desc := ""
		if len(args) > 1 {
			desc = strings.Join(args[1:], " ")
		}
		if err := backup.Create(desc); err != nil {
			ui.Fatal("%v", err)
		}
	case "restore":
		if len(args) < 2 {
			ui.Fatal("backup restore requires a timestamp")
		}
		if err := backup.Restore(args[1]); err != nil {
			ui.Fatal("%v", err)
		}
	default:
		ui.Fatal("Unknown backup command: %s", args[0])
	}
}

func handlePlan(args []string) {
	if len(args) == 0 {
		p, err := plan.UpdatePlan()
		if err != nil {
			ui.Fatal("Plan failed: %v", err)
		}
		p.Print()
		return
	}

	p, err := plan.InstallPlan(args[0])
	if err != nil {
		ui.Fatal("Plan failed: %v", err)
	}
	p.Print()
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
