// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"os/exec"
	"sync"
)

// toolBinary resolves a tool to a fixed absolute path ONCE, so every command runs that
// resolved binary rather than whatever a (possibly attacker-mutable) PATH maps the bare
// name to at call time — SonarCloud go:S4036, answered the same way cmd/obkversion does.
// It falls back to the bare name when the tool is not on PATH, so the failure surfaces as
// the command's own "executable file not found" rather than as a confusing empty path.
func toolBinary(name string, cache *sync.Once, dst *string) string {
	cache.Do(func() {
		if p, err := exec.LookPath(name); err == nil {
			*dst = p
			return
		}
		*dst = name
	})
	return *dst
}

var (
	goOnce, gitOnce sync.Once
	goPath, gitPath string
)

// goBinary is the resolved `go` command. See [toolBinary].
func goBinary() string { return toolBinary("go", &goOnce, &goPath) }

// gitBinary is the resolved `git` command. See [toolBinary].
func gitBinary() string { return toolBinary("git", &gitOnce, &gitPath) }
