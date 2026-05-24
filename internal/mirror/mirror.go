package mirror

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aapollo/up/internal/ui"
)

// Default mirrors
var mirrors = []string{
	"https://mirror.rackspace.com/archlinux",
	"https://archlinux.thaller.ws",
	"https://mirror.lty.me/archlinux",
}

// Package represents a downloadable package from official repos.
type Package struct {
	Name        string
	Version     string
	Repo        string
	Arch        string
	Filename    string
	URL         string
	SHA256      string
	Depends     []string
	Description string
	Size        int64
}

// FindPackage searches all repos for a package and returns download info.
func FindPackage(name string) (*Package, error) {
	// Sync db if needed
	if err := syncDBs(); err != nil {
		return nil, err
	}

	// Search in synced databases
	for _, repo := range []string{"core", "extra", "community", "multilib"} {
		pkg, err := searchRepoDB(repo, name)
		if err == nil && pkg != nil {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("package %s not found in official repositories", name)
}

// Download fetches a package from the fastest mirror.
func Download(pkg *Package, dest string) error {
	os.MkdirAll(filepath.Dir(dest), 0755)

	// Try each mirror
	var lastErr error
	for _, mirror := range mirrors {
		url := fmt.Sprintf("%s/%s/os/x86_64/%s", mirror, pkg.Repo, pkg.Filename)
		ui.Step("Downloading from %s...", mirror)

		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}

		// Write to file with progress
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer f.Close()

		written, err := io.Copy(f, resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		ui.Success("Downloaded %s (%.1f MB)", pkg.Filename, float64(written)/(1024*1024))
		return nil
	}

	return fmt.Errorf("failed to download from all mirrors: %v", lastErr)
}

// syncDBs downloads fresh repo databases if stale.
func syncDBs() error {
	dbDir := dbPath()
	os.MkdirAll(dbDir, 0755)

	for _, repo := range []string{"core", "extra"} {
		dbFile := filepath.Join(dbDir, fmt.Sprintf("%s.db", repo))
		// Check if db is fresh (< 1 hour old)
		if info, err := os.Stat(dbFile); err == nil {
			if time.Since(info.ModTime()) < time.Hour {
				continue
			}
		}

		ui.Step("Syncing %s database...", repo)
		if err := downloadDB(repo, dbFile); err != nil {
			ui.Error("Failed to sync %s: %v", repo, err)
		}
	}
	return nil
}

// downloadDB fetches a repo database.
func downloadDB(repo, dest string) error {
	for _, mirror := range mirrors {
		url := fmt.Sprintf("%s/%s/os/x86_64/%s.db", mirror, repo, repo)
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, resp.Body)
		return err
	}
	return fmt.Errorf("failed to download %s.db", repo)
}

// searchRepoDB searches a repo database for a package.
func searchRepoDB(repo, name string) (*Package, error) {
	dbFile := filepath.Join(dbPath(), fmt.Sprintf("%s.db", repo))
	f, err := os.Open(dbFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// .db files are tar.gz
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Package entries are directories like "pkgname-version/"
		if header.Typeflag == tar.TypeDir {
			parts := strings.Split(header.Name, "-")
			if len(parts) >= 2 {
				pkgName := strings.Join(parts[:len(parts)-2], "-")
				if pkgName == name {
					// Found it - read desc file
					return parseDesc(tr, repo, name)
				}
			}
		}
	}
	return nil, nil
}

// parseDesc extracts package info from the desc file in the db.
func parseDesc(tr *tar.Reader, repo, name string) (*Package, error) {
	pkg := &Package{Name: name, Repo: repo}
	// Simplified: in reality we'd parse the desc file format
	// For now return basic info
	pkg.Filename = fmt.Sprintf("%s-1-x86_64.pkg.tar.zst", name)
	return pkg, nil
}

func dbPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "up", "sync")
}
