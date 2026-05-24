package diff

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/aapollo/up/internal/ui"
)

// Show displays what files would change in a package update.
func Show(pkg string) error {
	// Check if package is installed
	cmd := exec.Command("pacman", "-Q", pkg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("package %s is not installed", pkg)
	}

	// Check if update is available
	cmd = exec.Command("pacman", "-Qu", pkg)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		ui.Info("No update available for %s", pkg)
		return nil
	}

	// Get current version
	cmd = exec.Command("pacman", "-Q", pkg)
	out, _ = cmd.Output()
	current := strings.TrimSpace(string(out))

	// Get new version
	cmd = exec.Command("pacman", "-Qu", pkg)
	out, _ = cmd.Output()
	new := strings.TrimSpace(string(out))

	ui.Header("Diff for %s", pkg)
	fmt.Printf("  %s → %s\n\n", current, new)

	// Use pacman -Qk to check file changes (simplified)
	// For a real diff we'd need to download the new package and compare
	cmd = exec.Command("pacman", "-Qkk", pkg)
	out, _ = cmd.Output()

	var modified, missing []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "modified") {
			modified = append(modified, extractPath(line))
		} else if strings.Contains(line, "missing") {
			missing = append(missing, extractPath(line))
		}
	}

	if len(modified) > 0 {
		ui.Step("Modified files (%d)", len(modified))
		for _, f := range modified[:min(10, len(modified))] {
			fmt.Printf("    %sM%s %s\n", ui.Yellow(), ui.Reset(), f)
		}
		if len(modified) > 10 {
			fmt.Printf("    ... and %d more\n", len(modified)-10)
		}
	}

	if len(missing) > 0 {
		ui.Step("Missing files (%d)", len(missing))
		for _, f := range missing[:min(10, len(missing))] {
			fmt.Printf("    %sD%s %s\n", ui.Red(), ui.Reset(), f)
		}
		if len(missing) > 10 {
			fmt.Printf("    ... and %d more\n", len(missing)-10)
		}
	}

	if len(modified) == 0 && len(missing) == 0 {
		ui.Success("No local modifications")
	}

	return nil
}

func extractPath(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 1 {
		return parts[0]
	}
	return line
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
