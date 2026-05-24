package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Dir returns the cache directory for up.
func Dir() string {
	dir := os.Getenv("UP_CACHE_DIR")
	if dir != "" {
		return dir
	}
	xdg := os.Getenv("XDG_CACHE_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".cache")
	}
	return filepath.Join(xdg, "up")
}

// BuildCacheDir returns the build cache directory.
func BuildCacheDir() string {
	return filepath.Join(Dir(), "builds")
}

// BinaryCacheDir returns the binary cache directory.
func BinaryCacheDir() string {
	return filepath.Join(Dir(), "binaries")
}

// HashBuildDir computes a hash of a build directory's PKGBUILD and .SRCINFO.
func HashBuildDir(buildDir string) (string, error) {
	pkgbuildPath := filepath.Join(buildDir, "PKGBUILD")
	srcinfoPath := filepath.Join(buildDir, ".SRCINFO")

	h := sha256.New()

	if data, err := os.ReadFile(pkgbuildPath); err == nil {
		h.Write(data)
	}
	if data, err := os.ReadFile(srcinfoPath); err == nil {
		h.Write(data)
	}

	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}

// FindCachedBinary looks for a cached binary package.
func FindCachedBinary(pkgName, buildHash string) (string, bool) {
	cacheDir := BinaryCacheDir()
	pattern := filepath.Join(cacheDir, fmt.Sprintf("%s-%s-*.pkg.tar.zst", pkgName, buildHash))

	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", false
	}
	return matches[0], true
}

// CacheBinary copies a built package to the binary cache.
func CacheBinary(pkgPath, pkgName, buildHash string) error {
	cacheDir := BinaryCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	base := filepath.Base(pkgPath)
	// Insert hash into filename
	parts := strings.Split(base, "-")
	if len(parts) >= 2 {
		parts = append(parts[:len(parts)-1], buildHash+".pkg.tar.zst")
		base = strings.Join(parts, "-")
	}

	dest := filepath.Join(cacheDir, base)
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0644)
}

// Clean removes old cached builds, keeping the last N.
func Clean(keep int) error {
	cacheDir := BinaryCacheDir()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(entries) <= keep {
		return nil
	}

	// Sort by modification time (oldest first)
	type fileInfo struct {
		name string
		mod  os.FileInfo
	}
	var files []fileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: entry.Name(), mod: info})
	}

	// Simple bubble sort by mod time
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].mod.ModTime().After(files[j].mod.ModTime()) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Remove oldest
	for i := 0; i < len(files)-keep; i++ {
		os.Remove(filepath.Join(cacheDir, files[i].name))
	}
	return nil
}

// Size returns the total size of the cache in bytes.
func Size() (int64, error) {
	var total int64
	cacheDir := Dir()
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}
