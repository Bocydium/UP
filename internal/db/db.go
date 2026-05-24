package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PackageRecord tracks an installed package.
type PackageRecord struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Repo      string    `json:"repo"`
	Files     []string  `json:"files"`
	Installed time.Time `json:"installed"`
}

// DB is the local package database.
type DB struct {
	path     string
	Packages map[string]PackageRecord `json:"packages"`
}

// Open loads the package database.
func Open() (*DB, error) {
	d := &DB{
		path:     filepath.Join(dataDir(), "db.json"),
		Packages: make(map[string]PackageRecord),
	}
	data, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			return d, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, d); err != nil {
		return nil, fmt.Errorf("corrupt db: %w", err)
	}
	return d, nil
}

// Save writes the database to disk.
func (d *DB) Save() error {
	os.MkdirAll(filepath.Dir(d.path), 0755)
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.path, data, 0644)
}

// Install records a package installation.
func (d *DB) Install(name, version, repo string, files []string) {
	d.Packages[name] = PackageRecord{
		Name:      name,
		Version:   version,
		Repo:      repo,
		Files:     files,
		Installed: time.Now(),
	}
}

// Remove deletes a package record.
func (d *DB) Remove(name string) {
	delete(d.Packages, name)
}

// IsInstalled checks if a package is in the database.
func (d *DB) IsInstalled(name string) bool {
	_, ok := d.Packages[name]
	return ok
}

// Get returns a package record.
func (d *DB) Get(name string) (PackageRecord, bool) {
	p, ok := d.Packages[name]
	return p, ok
}

func dataDir() string {
	dir := os.Getenv("UP_DATA_DIR")
	if dir != "" {
		return dir
	}
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdg, "up")
}
