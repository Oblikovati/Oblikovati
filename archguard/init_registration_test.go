// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Registries must be assembled by explicit construction at a composition root,
// not by init() side effects (#1617, audit B6): init()-time registration makes
// correctness depend on a binary's import list (documents silently opened as
// stubs when a blank import was forgotten) and blocks constructing a minimal
// set in tests. The contentFactories migration removed the pattern from
// model/doc/compdef/drawing; this guard keeps it out everywhere else too,
// with a SHRINK-ONLY allowlist for the registries whose migration is pending.

// initRegistrationPattern matches an init() whose body performs a registration
// call — the import-linkage wiring B6 retires.
var initRegistrationPattern = regexp.MustCompile(`func init\(\) \{[^}]*[rR]egister`)

// pendingInitRegistrations are files whose init()-time registrations await
// their own migration PR (#1617 lands one registry per PR). SHRINK-ONLY:
// migrate the registry to an explicit default-set constructor and delete the
// entry. Never add to it.
var pendingInitRegistrations = map[string]struct{}{
	"model/sketch/serialize_codecs_3d.go":             {}, // sketch entity/constraint codecs (#1624/#1625) → follow-up
	"model/sketch/serialize_codecs_curves.go":         {},
	"model/sketch/serialize_codecs_extras.go":         {},
	"model/sketch/serialize_codecs_constraints.go":    {},
	"model/sketch/serialize_codecs_constraints_3d.go": {},
}

func TestNoInitTimeRegistrations(t *testing.T) {
	offenders := initRegistrationSources(t)
	seen := map[string]struct{}{}
	for _, f := range offenders {
		seen[f] = struct{}{}
		if _, pending := pendingInitRegistrations[f]; !pending {
			t.Errorf("%s registers via init() — assemble the registry in an explicit default-set constructor wired at the composition root instead (#1617, audit B6)", f)
		}
	}
	for f := range pendingInitRegistrations {
		if _, ok := seen[f]; !ok {
			t.Errorf("pendingInitRegistrations entry %q is stale — the file no longer registers via init(); delete the entry (shrink-only, #1617)", f)
		}
	}
}

// initRegistrationSources walks the module's first-party sources — INCLUDING
// model/sketch and the head submodule, which other guards prune — and returns
// the files whose init() performs a registration.
func initRegistrationSources(t *testing.T) []string {
	t.Helper()
	var offenders []string
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Dot-directories cover .git AND .claude (agent worktrees carry full repo copies).
			if strings.HasPrefix(info.Name(), ".") && info.Name() != ".." && info.Name() != "." {
				return filepath.SkipDir
			}
			switch info.Name() {
			case "experiments", "test-utilities", "architecture", "testdata", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if initRegistrationPattern.Match(src) {
			// ToSlash so the allowlist's forward-slash keys match on Windows too.
			offenders = append(offenders, strings.TrimPrefix(filepath.ToSlash(path), "../"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking module sources: %v", err)
	}
	return offenders
}
