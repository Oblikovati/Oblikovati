// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati/kernel/ops"
)

// Every boolean (material-changing) feature must expose its tool and operation via
// [ToolFeature], so a pattern/mirror replicates it correctly — re-applying the source's
// cut/join at each occurrence instead of copying the whole body (the bug that split a
// patterned hole/blade into N solids). These compile-time assertions fail the build if any
// loses the contract; the runtime replication itself is covered by the extrude pattern tests
// (TestPatternOfCutKeepsOneBody / TestPatternOfJoinMergesIntoOneBody) and the fan loft e2e.
var (
	_ ToolFeature = (*ExtrudeFeature)(nil)
	_ ToolFeature = (*RevolveFeature)(nil)
	_ ToolFeature = (*CoilFeature)(nil)
	_ ToolFeature = (*RibFeature)(nil)
	_ ToolFeature = (*EmbossFeature)(nil)
	_ ToolFeature = (*LoftFeature)(nil)
	_ ToolFeature = (*SweepFeature)(nil)
	_ ToolFeature = (*HoleFeature)(nil) // a hole caches its drill cylinder as the replicable tool
)

// Boss geometry still defers, so it has no tool body yet, but it must still report its operation
// so a pattern of a boss is treated as a (currently empty) join rather than a whole-body copy.
var _ OperationalFeature = (*BossFeature)(nil)

// TestEveryBooleanFeatureExposesItsTool documents the contract above at test time too: each
// listed feature reports its operation through OperationalFeature (extrude here as the
// reference), so operationOf resolves it for the pattern engine rather than defaulting to a
// new-body copy.
func TestEveryBooleanFeatureExposesItsTool(t *testing.T) {
	var f Feature = &ExtrudeFeature{def: &ExtrudeDefinition{Operation: ops.Cut}}
	tf, ok := f.(ToolFeature)
	if !ok {
		t.Fatal("ExtrudeFeature must implement ToolFeature")
	}
	if tf.Operation() != ops.Cut {
		t.Errorf("Operation() = %v, want Cut", tf.Operation())
	}
}
