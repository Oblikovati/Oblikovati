// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestAssemblyDrivePreviewOverWire adds a rotational joint between two boxes' shared edge axis,
// then drives it through a range over the wire and checks the frames advance the moved
// component's rotation — the F03 acceptance over the router (#366).
func TestAssemblyDrivePreviewOverWire(t *testing.T) {
	t.Parallel()
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	edge := boxEdgeKey(t, occs[0])

	var added wire.AssemblyJointResult
	call(t, r, s, "assemblyJoints.addRotational", mustJSON(t, wire.AddJointArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: edge},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: edge},
	}), &added)

	span := math.Pi / 2
	var res wire.DriveResult
	call(t, r, s, "assemblyDrive.preview", mustJSON(t, wire.DriveJointArgs{
		Joint: added.Joint.ID,
		Settings: wire.DriveSettingsDTO{
			Variable: "angular", Start: 0, End: span, Step: span / 4,
		},
	}), &res)

	if len(res.Frames) != 5 {
		t.Fatalf("frames = %d, want 5", len(res.Frames))
	}
	for _, f := range res.Frames {
		if len(f.Placements) != 1 || f.Placements[0].Occurrence != occs[1].ID() {
			t.Fatalf("frame placements = %+v, want the one moved occurrence", f.Placements)
		}
	}
	// The moved frame's orientation sweeps through the driven span (measured axis-independently
	// from the relative rotation between the first and last frame).
	first := res.Frames[0].Placements[0].Transform
	last := res.Frames[len(res.Frames)-1].Placements[0].Transform
	if delta := rotationAngleBetween(first, last); math.Abs(delta-span) > 1e-3 {
		t.Errorf("driven rotation = %.4f rad, want ≈ %.4f", delta, span)
	}
}

// TestAssemblyDriveUndrivableJointRejected checks a rigid joint (0 DOF) is a clean error.
func TestAssemblyDriveUndrivableJointRejected(t *testing.T) {
	t.Parallel()
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	edge := boxEdgeKey(t, occs[0])

	var added wire.AssemblyJointResult
	call(t, r, s, "assemblyJoints.addRigid", mustJSON(t, wire.AddJointArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: edge},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: edge},
	}), &added)

	args := mustJSON(t, wire.DriveJointArgs{Joint: added.Joint.ID, Settings: wire.DriveSettingsDTO{End: 1, Step: 0.5}})
	if _, err := r.Handle(s, "assemblyDrive.preview", []byte(args)); err == nil {
		t.Error("driving a rigid joint should fail (not drivable)")
	}
}

// rotationAngleBetween returns the angle of the relative rotation between two rigid frames,
// independent of the (unknown) joint axis: acos((trace(Aᵀ·B)−1)/2) over their 3×3 rotation
// parts.
func rotationAngleBetween(a, b types.Matrix) float64 {
	trace := 0.0
	for i := range 3 {
		for j := range 3 {
			trace += a.At(i, j) * b.At(i, j) // Σ a[i][j]·b[i][j] = trace(A·Bᵀ)
		}
	}
	c := (trace - 1) / 2
	if c > 1 {
		c = 1
	} else if c < -1 {
		c = -1
	}
	return math.Acos(c)
}
