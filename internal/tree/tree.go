package tree

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/aapollo/up/internal/ui"
)

// Node represents a package in the dependency tree.
type Node struct {
	Name     string
	Version  string
	Children []*Node
	Depth    int
}

// Show displays a dependency tree for a package.
func Show(pkg string) error {
	cmd := exec.Command("pacman", "-Si", pkg)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("package %s not found", pkg)
	}

	// Parse dependencies from pacman -Si
	var deps []string
	inDepends := false
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Depends On") {
			inDepends = true
			depLine := strings.TrimPrefix(line, "Depends On")
			depLine = strings.TrimPrefix(depLine, ":")
			depLine = strings.TrimSpace(depLine)
			if depLine != "None" {
				deps = append(deps, strings.Fields(depLine)...)
			}
		} else if inDepends && strings.HasPrefix(line, "              ") {
			// Continuation line
			depLine := strings.TrimSpace(line)
			if depLine != "None" {
				deps = append(deps, strings.Fields(depLine)...)
			}
		} else if inDepends && !strings.HasPrefix(line, "              ") {
			inDepends = false
		}
	}

	ui.Header("Dependency tree: %s", pkg)
	fmt.Printf("\n%s\n", pkg)

	// Show first level deps
	for i, dep := range deps {
		isLast := i == len(deps)-1
		prefix := "├──"
		if isLast {
			prefix = "└──"
		}
		// Clean up dependency name (remove version constraints)
		cleanDep := strings.Split(dep, ">=")[0]
		cleanDep = strings.Split(cleanDep, "=")[0]
		cleanDep = strings.Split(cleanDep, "<")[0]
		cleanDep = strings.TrimSpace(cleanDep)

		fmt.Printf("%s %s\n", prefix, cleanDep)
	}

	if len(deps) == 0 {
		ui.Info("No dependencies")
	}

	return nil
}
