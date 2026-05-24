package plan

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/aapollo/up/internal/aur"
	"github.com/aapollo/up/internal/cache"
	"github.com/aapollo/up/internal/pacman"
	"github.com/aapollo/up/internal/ui"
)

// Plan represents what would happen during an operation.
type Plan struct {
	OfficialInstalls []string
	OfficialUpdates  []string
	AURBuilds        []AURBuild
	CachedInstalls   []string
	Removals         []string
	TotalDownloadMB  float64
	TotalBuildTime   string
}

// AURBuild represents a planned AUR build.
type AURBuild struct {
	Name      string
	Version   string
	FromCache bool
	BuildTime string // estimated
}

// InstallPlan generates a plan for installing a package.
func InstallPlan(pkg string) (*Plan, error) {
	plan := &Plan{}

	// Check official repos
	if pacman.InOfficialRepo(pkg) {
		if pacman.IsInstalled(pkg) {
			// Check if update available
			cmd := exec.Command("pacman", "-Qu", pkg)
			out, _ := cmd.Output()
			if len(out) > 0 {
				plan.OfficialUpdates = append(plan.OfficialUpdates, pkg)
			}
		} else {
			plan.OfficialInstalls = append(plan.OfficialInstalls, pkg)
		}
		return plan, nil
	}

	// Check AUR
	info, err := aur.Search(pkg)
	if err != nil || info == nil {
		return nil, fmt.Errorf("package %s not found", pkg)
	}

	// Check cache
	buildHash := ""
	if hash, err := cache.HashBuildDir(""); err == nil {
		buildHash = hash
	}
	fromCache := false
	if buildHash != "" {
		if _, found := cache.FindCachedBinary(info.Name, buildHash); found {
			fromCache = true
		}
	}

	plan.AURBuilds = append(plan.AURBuilds, AURBuild{
		Name:      info.Name,
		Version:   info.Version,
		FromCache: fromCache,
		BuildTime: estimateBuildTime(info.Name),
	})

	return plan, nil
}

// UpdatePlan generates a plan for updating all packages.
func UpdatePlan() (*Plan, error) {
	plan := &Plan{}

	// Check official repo updates
	cmd := exec.Command("pacman", "-Qu")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				plan.OfficialUpdates = append(plan.OfficialUpdates, fields[0])
			}
		}
	}

	// Check AUR updates
	pkgs, err := pacman.GetInstalledAURPackages()
	if err == nil {
		for _, pkg := range pkgs {
			info, err := aur.Info(pkg)
			if err != nil {
				continue
			}
			cmd := exec.Command("pacman", "-Q", pkg)
			out, _ := cmd.Output()
			installed := strings.TrimSpace(string(out))
			parts := strings.Fields(installed)
			if len(parts) >= 2 && parts[1] != info.Version {
				// Check cache
				buildHash, _ := cache.HashBuildDir("")
				fromCache := false
				if buildHash != "" {
					if _, found := cache.FindCachedBinary(pkg, buildHash); found {
						fromCache = true
					}
				}
				plan.AURBuilds = append(plan.AURBuilds, AURBuild{
					Name:      pkg,
					Version:   info.Version,
					FromCache: fromCache,
					BuildTime: estimateBuildTime(pkg),
				})
			}
		}
	}

	return plan, nil
}

// Print displays the plan in a readable format.
func (p *Plan) Print() {
	ui.Header("Plan")

	if len(p.OfficialInstalls) > 0 {
		ui.Step("Install from official repos (%d)", len(p.OfficialInstalls))
		ui.PackageList(p.OfficialInstalls)
	}

	if len(p.OfficialUpdates) > 0 {
		ui.Step("Update from official repos (%d)", len(p.OfficialUpdates))
		ui.PackageList(p.OfficialUpdates)
	}

	if len(p.AURBuilds) > 0 {
		cached := 0
		for _, b := range p.AURBuilds {
			if b.FromCache {
				cached++
			}
		}
		if cached == len(p.AURBuilds) {
			ui.Step("AUR packages (%d, all cached)", len(p.AURBuilds))
		} else if cached > 0 {
			ui.Step("AUR packages (%d, %d cached)", len(p.AURBuilds), cached)
		} else {
			ui.Step("AUR packages to build (%d)", len(p.AURBuilds))
		}
		for _, b := range p.AURBuilds {
			if b.FromCache {
				fmt.Printf("    %s (cached)\n", b.Name)
			} else {
				fmt.Printf("    %s (~%s)\n", b.Name, b.BuildTime)
			}
		}
	}

	if len(p.CachedInstalls) > 0 {
		ui.Step("Install from cache (%d)", len(p.CachedInstalls))
		ui.PackageList(p.CachedInstalls)
	}

	if len(p.Removals) > 0 {
		ui.Step("Remove (%d)", len(p.Removals))
		ui.PackageList(p.Removals)
	}

	if p.isEmpty() {
		ui.Info("Nothing to do")
	}
}

func (p *Plan) isEmpty() bool {
	return len(p.OfficialInstalls) == 0 &&
		len(p.OfficialUpdates) == 0 &&
		len(p.AURBuilds) == 0 &&
		len(p.CachedInstalls) == 0 &&
		len(p.Removals) == 0
}

func estimateBuildTime(pkg string) string {
	// Simple heuristic based on package name patterns
	if strings.Contains(pkg, "linux") || strings.Contains(pkg, "firefox") || strings.Contains(pkg, "chromium") {
		return "10-30 min"
	}
	if strings.Contains(pkg, "electron") || strings.Contains(pkg, "jdk") {
		return "5-15 min"
	}
	return "1-5 min"
}
