package pacman

import (
	"os"
	"os/exec"
	"strings"

	"github.com/aapollo/up/internal/cli"
)

// PackageInfo holds metadata about a package.
type PackageInfo struct {
	Name        string
	Version     string
	Repo        string
	Description string
	URL         string
	License     string
	Depends     []string
}

// IsInstalled checks if a package is already installed.
func IsInstalled(pkg string) bool {
	cmd := exec.Command("pacman", "-Q", pkg)
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err == nil
}

// InOfficialRepo checks if a package exists in official repos.
func InOfficialRepo(pkg string) bool {
	cmd := exec.Command("pacman", "-Si", pkg)
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err == nil
}

// Install installs a package from official repos using pacman.
func Install(pkg string, flags cli.Flags) error {
	args := []string{"-S", "--needed"}
	if flags.NoConfirm {
		args = append(args, "--noconfirm")
	}
	args = append(args, pkg)

	cmd := exec.Command("sudo", append([]string{"pacman"}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Remove removes a package and its unused dependencies.
func Remove(pkg string, flags cli.Flags) error {
	args := []string{"-Rns"}
	if flags.NoConfirm {
		args = append(args, "--noconfirm")
	}
	args = append(args, pkg)

	cmd := exec.Command("sudo", append([]string{"pacman"}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Update updates all packages from official repos.
func Update() error {
	cmd := exec.Command("sudo", "pacman", "-Syu")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Search searches official repos for packages matching query.
func Search(query string) ([]PackageInfo, error) {
	cmd := exec.Command("pacman", "-Ss", query)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var results []PackageInfo
	lines := strings.Split(string(out), "\n")
	for i := 0; i < len(lines); i += 2 {
		if i+1 >= len(lines) {
			break
		}
		nameLine := lines[i]
		descLine := lines[i+1]

		parts := strings.SplitN(nameLine, " ", 2)
		if len(parts) < 2 {
			continue
		}
		repoName := strings.SplitN(parts[0], "/", 2)
		if len(repoName) != 2 {
			continue
		}

		results = append(results, PackageInfo{
			Repo:        repoName[0],
			Name:        repoName[1],
			Version:     strings.TrimSpace(parts[1]),
			Description: strings.TrimSpace(descLine),
		})
	}
	return results, nil
}

// Info gets detailed info about a package from official repos.
func Info(pkg string) (PackageInfo, error) {
	cmd := exec.Command("pacman", "-Si", pkg)
	out, err := cmd.Output()
	if err != nil {
		return PackageInfo{}, err
	}

	info := PackageInfo{Name: pkg}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Repository") {
			info.Repo = strings.TrimSpace(strings.TrimPrefix(line, "Repository"))
			info.Repo = strings.TrimPrefix(info.Repo, ": ")
		} else if strings.HasPrefix(line, "Version") {
			info.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version"))
			info.Version = strings.TrimPrefix(info.Version, ": ")
		} else if strings.HasPrefix(line, "Description") {
			info.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description"))
			info.Description = strings.TrimPrefix(info.Description, ": ")
		} else if strings.HasPrefix(line, "URL") {
			info.URL = strings.TrimSpace(strings.TrimPrefix(line, "URL"))
			info.URL = strings.TrimPrefix(info.URL, ": ")
		} else if strings.HasPrefix(line, "Licenses") {
			info.License = strings.TrimSpace(strings.TrimPrefix(line, "Licenses"))
			info.License = strings.TrimPrefix(info.License, ": ")
		} else if strings.HasPrefix(line, "Depends On") {
			deps := strings.TrimSpace(strings.TrimPrefix(line, "Depends On"))
			deps = strings.TrimPrefix(deps, ": ")
			info.Depends = strings.Fields(deps)
		}
	}
	return info, nil
}

// GetInstalledAURPackages returns a list of packages installed from AUR.
func GetInstalledAURPackages() ([]string, error) {
	cmd := exec.Command("pacman", "-Qm")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pkgs []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			pkgs = append(pkgs, fields[0])
		}
	}
	return pkgs, nil
}
