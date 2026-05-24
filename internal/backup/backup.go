package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aapollo/up/internal/ui"
)

// Snapshot represents a backup of installed packages.
type Snapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	Official    []string  `json:"official"`
	AUR         []string  `json:"aur"`
	Description string    `json:"description"`
}

// Dir returns the backup directory.
func Dir() string {
	dir := os.Getenv("UP_DATA_DIR")
	if dir != "" {
		return filepath.Join(dir, "backups")
	}
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdg, "up", "backups")
}

// Create saves a snapshot of currently installed packages.
func Create(desc string) error {
	os.MkdirAll(Dir(), 0755)

	// Get official packages
	cmd := exec.Command("pacman", "-Qnq")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list official packages: %w", err)
	}
	var official []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			official = append(official, line)
		}
	}

	// Get AUR packages
	cmd = exec.Command("pacman", "-Qmq")
	out, err = cmd.Output()
	var aur []string
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				aur = append(aur, line)
			}
		}
	}

	snap := Snapshot{
		Timestamp:   time.Now(),
		Official:    official,
		AUR:         aur,
		Description: desc,
	}

	filename := fmt.Sprintf("snapshot-%s.json", snap.Timestamp.Format("20060102-150405"))
	path := filepath.Join(Dir(), filename)

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	ui.Success("Backup saved: %s (%d official, %d AUR)", filename, len(official), len(aur))
	return nil
}

// List shows all available backups.
func List() error {
	entries, err := os.ReadDir(Dir())
	if err != nil {
		if os.IsNotExist(err) {
			ui.Info("No backups found")
			return nil
		}
		return err
	}

	ui.Header("Backups")
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(Dir(), entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var snap Snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			continue
		}
		fmt.Printf("  %s  %d official, %d AUR  %s\n",
			snap.Timestamp.Format("2006-01-02 15:04"),
			len(snap.Official),
			len(snap.AUR),
			snap.Description,
		)
	}
	return nil
}

// Restore restores packages from a backup.
func Restore(timestamp string) error {
	// Find matching backup
	entries, err := os.ReadDir(Dir())
	if err != nil {
		return err
	}

	var targetPath string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), timestamp) {
			targetPath = filepath.Join(Dir(), entry.Name())
			break
		}
	}

	if targetPath == "" {
		return fmt.Errorf("no backup found matching %q", timestamp)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return err
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}

	ui.Header("Restoring backup from %s", snap.Timestamp.Format("2006-01-02 15:04"))
	ui.Step("Would reinstall %d official and %d AUR packages", len(snap.Official), len(snap.AUR))
	ui.Info("Use `up inst <pkg>` to install individual packages from this backup")

	return nil
}
