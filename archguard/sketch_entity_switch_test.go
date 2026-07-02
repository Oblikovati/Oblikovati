// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Sketch entities name their own kind, shape, trim, and rebind behavior
// (#1624, audit I1) — a consumer outside model/sketch that re-derives what an
// entity is with a `case *sketch.X:` type switch re-creates the drift class
// that made saves fail at runtime (#1416) and left kinds enumerable-but-not-
// editable (#1574). The forbidden type list is parsed from model/sketch's
// Kind() declarations, so a new entity kind extends this guard automatically.
// Constraint-type switches are a separate finding (I2, #1625) and are not
// covered here; single type assertions (extracting one typed pick) are fine.

// entityKindReceiverPattern extracts the receiver type of every Kind() method
// in model/sketch — the closed set of entity types.
var entityKindReceiverPattern = regexp.MustCompile(`func \(\w+ \*(\w+)\) Kind\(\) EntityKind`)

func TestNoSketchEntityTypeSwitchesOutsideModelSketch(t *testing.T) {
	entityTypes := sketchEntityTypeNames(t)
	if len(entityTypes) < 30 {
		t.Fatalf("parsed only %d entity types from model/sketch (want the full ~35) — did Kind() move out of entity_kind.go?", len(entityTypes))
	}
	entityCase := regexp.MustCompile(`case [^:\n]*\*sketch\.(` + strings.Join(entityTypes, "|") + `)\b`)
	for _, offender := range goSourcesMatching(t, entityCase) {
		t.Errorf("%s switches on a sketch entity type — use the entity's capability (Kind/ShapePoints/TrimCurveAt/RebindProjection/RadiusDOF) or a single typed assertion instead (#1624, audit I1)", offender)
	}
}

// sketchEntityTypeNames parses the Kind() receivers out of model/sketch.
func sketchEntityTypeNames(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("../model/sketch/entity_kind.go")
	if err != nil {
		t.Fatalf("reading model/sketch/entity_kind.go: %v", err)
	}
	var out []string
	for _, m := range entityKindReceiverPattern.FindAllStringSubmatch(string(src), -1) {
		out = append(out, m[1])
	}
	return out
}

// goSourcesMatching walks the module's non-test sources outside model/sketch
// and returns the files matching the pattern.
func goSourcesMatching(t *testing.T, pattern *regexp.Regexp) []string {
	t.Helper()
	var offenders []string
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return skipNonModuleDirs(path, info.Name())
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if pattern.Match(src) {
			// ToSlash so reported paths (and any caller matching on them) are
			// separator-stable on Windows.
			offenders = append(offenders, strings.TrimPrefix(filepath.ToSlash(path), "../"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking module sources: %v", err)
	}
	return offenders
}

// skipNonModuleDirs prunes the walk to this module's first-party sources:
// model/sketch owns its own switches, and generated/vendored/experimental
// trees are not subject to the seam.
func skipNonModuleDirs(path, name string) error {
	if strings.HasSuffix(filepath.ToSlash(path), "model/sketch") {
		return filepath.SkipDir
	}
	switch name {
	case ".git", "experiments", "test-utilities", "architecture", "testdata", "node_modules":
		return filepath.SkipDir
	}
	return nil
}
