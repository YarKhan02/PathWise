// Package project resolves what is being analyzed: the module root, the git
// revision, and whether the working tree is clean.
//
// Revision identity matters more than it appears. A scan recorded against a
// dirty tree cannot be meaningfully compared to another revision, so the
// dirty flag is stored and surfaced rather than ignored.
package project

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Project is a resolved analysis target.
type Project struct {
	// Root is the absolute path to the directory containing go.mod.
	Root string

	// ModulePath is the module's import path from go.mod.
	ModulePath string

	// GitRevision is the resolved commit SHA, or "" outside a repository.
	GitRevision string

	// GitDirty is true when tracked files have uncommitted changes.
	GitDirty bool
}

// Resolve walks up from dir to find a go.mod, then reads git state.
func Resolve(dir string) (*Project, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	root, err := findModuleRoot(abs)
	if err != nil {
		return nil, err
	}

	modPath, err := readModulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}

	p := &Project{Root: root, ModulePath: modPath}
	p.GitRevision, p.GitDirty = gitState(root)
	return p, nil
}

func findModuleRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found at or above %s", start)
		}
		dir = parent
	}
}

func readModulePath(goMod string) (string, error) {
	f, err := os.Open(goMod)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "module"); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%s has no module directive", goMod)
}

// gitState returns the HEAD revision and whether the tree is dirty. Absence
// of git is not an error; it only means revision comparison is unavailable.
func gitState(root string) (rev string, dirty bool) {
	out, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return "", false
	}
	rev = strings.TrimSpace(out)

	status, err := runGit(root, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return rev, false
	}
	return rev, strings.TrimSpace(status) != ""
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var sb strings.Builder
	cmd.Stdout = &sb
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return sb.String(), nil
}
