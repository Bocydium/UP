package extract

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aapollo/up/internal/ui"
	"github.com/klauspost/compress/zstd"
)

// Install extracts a .pkg.tar.zst to the system root.
func Install(pkgPath string, root string) error {
	if root == "" {
		root = "/"
	}

	f, err := os.Open(pkgPath)
	if err != nil {
		return fmt.Errorf("open package: %w", err)
	}
	defer f.Close()

	// Decompress zstd
	zr, err := zstd.NewReader(f)
	if err != nil {
		return fmt.Errorf("zstd decompress: %w", err)
	}
	defer zr.Close()

	// Extract tar
	tr := tar.NewReader(zr)
	var filesInstalled int
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		// Skip .PKGINFO, .BUILDINFO, .MTREE
		if strings.HasPrefix(header.Name, ".") {
			continue
		}

		target := filepath.Join(root, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			filesInstalled++
		case tar.TypeSymlink:
			os.Remove(target) // Remove existing
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}

	ui.Success("Installed %d files", filesInstalled)
	return nil
}

// ListFiles returns all files in a package without extracting.
func ListFiles(pkgPath string) ([]string, error) {
	f, err := os.Open(pkgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	zr, err := zstd.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	tr := tar.NewReader(zr)
	var files []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeReg && !strings.HasPrefix(header.Name, ".") {
			files = append(files, header.Name)
		}
	}
	return files, nil
}
