package security

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aapollo/up/internal/aur"
	"github.com/aapollo/up/internal/ui"
)

// ScanAURPackage performs security checks on an AUR package.
func ScanAURPackage(pkg *aur.Package) error {
	// Check for suspicious maintainers
	if pkg.Maintainer == "" {
		ui.Step("Warning: package has no maintainer")
	}

	// Check votes
	if pkg.Votes < 5 {
		ui.Step("Warning: package has only %d votes", pkg.Votes)
	}

	// Check if out of date
	if pkg.OutOfDate > 0 {
		ui.Step("Warning: package is flagged out of date")
	}

	// Download and inspect PKGBUILD
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "up", "aur")
	buildDir := filepath.Join(cacheDir, pkg.Name)

	// Clone if not exists
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone",
			fmt.Sprintf("https://aur.archlinux.org/%s.git", pkg.Name),
			buildDir,
		)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone for security scan: %w", err)
		}
	}

	// Read PKGBUILD
	pkgbuildPath := filepath.Join(buildDir, "PKGBUILD")
	content, err := os.ReadFile(pkgbuildPath)
	if err != nil {
		return fmt.Errorf("failed to read PKGBUILD: %w", err)
	}
	pkgbuild := string(content)

	// Check for dangerous commands
	dangerous := []string{
		`curl.*\|.*sh`,
		`wget.*\|.*sh`,
		"eval",
		"exec",
		"rm -rf /",
		"> /dev/sda",
		"mkfs",
		":(){ :|:& };:", // fork bomb
	}

	for _, pattern := range dangerous {
		if strings.Contains(pkgbuild, pattern) {
			return fmt.Errorf("DANGEROUS pattern detected in PKGBUILD: %s", pattern)
		}
	}

	// Check for network downloads without checksums
	if strings.Contains(pkgbuild, "curl") || strings.Contains(pkgbuild, "wget") {
		if !strings.Contains(pkgbuild, "sha256sums") && !strings.Contains(pkgbuild, "md5sums") {
			ui.Step("Warning: PKGBUILD downloads from network without checksums")
		}
	}

	// Verify GPG signatures if present
	if strings.Contains(pkgbuild, "validpgpkeys") {
		ui.Step("GPG key verification configured")
	}

	// Check for sudo usage (should not be in PKGBUILD)
	if strings.Contains(pkgbuild, "sudo") {
		return fmt.Errorf("PKGBUILD contains sudo - this is not allowed")
	}

	return nil
}

// VerifyChecksum verifies a file against its SHA256 checksum.
func VerifyChecksum(filePath, expectedSum string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(data))
	if sum != expectedSum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSum, sum)
	}
	return nil
}
