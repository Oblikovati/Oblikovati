// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitChanges lists the paths that differ from a base revision, plus everything the
// working tree has staged, unstaged or left untracked. A local run must see the file
// the developer just wrote, not only what is committed.
type GitChanges struct {
	root string
	base string
}

// NewGitChanges returns a change lister for root, comparing against base (for example
// "origin/develop"). An empty base compares against the working tree only.
func NewGitChanges(root, base string) *GitChanges {
	return &GitChanges{root: root, base: base}
}

// ChangedPaths returns the repository-relative paths the change set touches.
func (g *GitChanges) ChangedPaths() ([]string, error) {
	set := map[string]bool{}
	args := [][]string{
		{"diff", "--name-only", "HEAD"},
		{"ls-files", "--others", "--exclude-standard"},
	}
	if g.base != "" {
		args = append(args, []string{"diff", "--name-only", g.base + "...HEAD"})
	}
	for _, a := range args {
		if err := g.collect(set, a); err != nil {
			return nil, err
		}
	}
	return sortedKeys(set), nil
}

// collect runs one git command and adds each output line to the set.
func (g *GitChanges) collect(set map[string]bool, args []string) error {
	// NOSONAR go:S4036 — gitBinary() has already resolved this through exec.LookPath. This is
	// developer tooling invoked from make in the developer's own shell; a PATH that can
	// substitute `git` owns that shell already.
	cmd := exec.Command(gitBinary(), args...) // NOSONAR
	cmd.Dir = g.root
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git %s in %s: %w", strings.Join(args, " "), g.root, err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			set[line] = true
		}
	}
	return nil
}
