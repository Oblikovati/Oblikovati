// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Sketch constraints name their own kind and related entities (#1625, audit
// I2) — a consumer outside model/sketch that re-derives what a constraint is
// with a `case *sketch.X:` type switch re-creates the drift class that left
// Symmetry enumerable but not creatable (#1574) and Equal3D failing every
// save at runtime (#1416 class). The forbidden type list is parsed from
// model/sketch's ConstraintKind() declarations, so a new constraint kind
// extends this guard automatically. Entity-type switches are guarded by
// sketch_entity_switch_test (#1624); single type assertions stay fine.

// constraintKindReceiverPattern extracts the receiver type of every
// ConstraintKind() method in model/sketch — the closed set of constraint types.
var constraintKindReceiverPattern = regexp.MustCompile(`func \(\w+ \*(\w+)\) ConstraintKind\(\) ConstraintKind`)

func TestNoSketchConstraintTypeSwitchesOutsideModelSketch(t *testing.T) {
	constraintTypes := sketchConstraintTypeNames(t)
	if len(constraintTypes) < 30 {
		t.Fatalf("parsed only %d constraint types from model/sketch (want the full ~37) — did ConstraintKind() move out of constraint_kind.go / constraint_kind_3d.go?", len(constraintTypes))
	}
	constraintCase := regexp.MustCompile(`case [^:\n]*\*sketch\.(` + strings.Join(constraintTypes, "|") + `)\b`)
	for _, offender := range goSourcesMatching(t, constraintCase) {
		t.Errorf("%s switches on a sketch constraint type — use the constraint's KindedConstraint capability (ConstraintKind/RelatedEntities) or a single typed assertion instead (#1625, audit I2)", offender)
	}
}

// sketchConstraintTypeNames parses the ConstraintKind() receivers out of
// model/sketch's kind declaration files.
func sketchConstraintTypeNames(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, file := range []string{"../model/sketch/constraint_kind.go", "../model/sketch/constraint_kind_3d.go"} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		for _, m := range constraintKindReceiverPattern.FindAllStringSubmatch(string(src), -1) {
			out = append(out, m[1])
		}
	}
	return out
}
