package flatpak

import (
	"os"
	"os/exec"
)

// Update updates all flatpak packages.
func Update() error {
	cmd := exec.Command("flatpak", "update", "-y")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
