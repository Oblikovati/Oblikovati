// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The from-to (kFromToExtent) and distance-from-face (kDistanceFromFaceExtent) extent modes: the
// model already carries FromPlane/ToPlane and the span geometry; these exercise the opregistry
// wiring that names the terminator plane(s). #1859.

// offsetXYPlaneRef adds a work plane parallel to the origin XY plane, offset by z (model cm), and
// returns its reference (e.g. "plane/0") — a stable, parallel terminator for the extent tests.
func offsetXYPlaneRef(t *testing.T, s *app.Session, z float64) string {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.ParseWorkRef("origin/plane/xy"), func() float64 { return z })
	def.Recompute()
	return string(wp.Key())
}

// TestExtrudeFromToBoundsBothPlanes: a from-to extrude of the 4×3 profile bounded by planes at
// z=0.5 and z=1.5 fills exactly that 1 cm-thick slab (12 cm³) — proving both FromPlane and ToPlane
// drive the span, not the sketch plane.
func TestExtrudeFromToBoundsBothPlanes(t *testing.T) {
	s := profiledPart(t)
	lo := offsetXYPlaneRef(t, s, 0.5)
	hi := offsetXYPlaneRef(t, s, 1.5)
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 0, "extent": "from-to", "fromFace": lo, "toFace": hi, "operation": "new",
	})
	if _, err := apply(t, s, "extrude", string(args)); err != nil {
		t.Fatalf("from-to extrude: %v", err)
	}
	got, want := bodyVolume(t, s), 4.0*3.0*1.0
	if math.Abs(got-want) > 1e-6*want {
		t.Errorf("from-to volume = %g, want %g (4×3 profile between z=0.5 and z=1.5)", got, want)
	}
}

// TestExtrudeDistanceFromFace: a distance-from-face extrude measures its 1 cm depth from a plane at
// z=0.5, so the solid spans z=0..1.5 (18 cm³) rather than z=0..1.0 (a plain distance would give 12).
func TestExtrudeDistanceFromFace(t *testing.T) {
	s := profiledPart(t)
	base := offsetXYPlaneRef(t, s, 0.5)
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 0, "extent": "distance-from-face", "toFace": base, "distance": "10 mm", "operation": "new",
	})
	if _, err := apply(t, s, "extrude", string(args)); err != nil {
		t.Fatalf("distance-from-face extrude: %v", err)
	}
	got, want := bodyVolume(t, s), 4.0*3.0*1.5
	if math.Abs(got-want) > 1e-6*want {
		t.Errorf("distance-from-face volume = %g, want %g (depth 1 cm from z=0.5)", got, want)
	}
}

// TestExtrudeFromToGeom: from-to naming BOTH terminators by GEOMETRY (centroid+normal) — the
// bottom and top faces of a seeded 2 cm box — builds the same 24 cm³ prism, exercising the
// geometric start/end resolvers (resolveStartPlane / resolveEndPlane via fromFaceGeom/toFaceGeom).
func TestExtrudeFromToGeom(t *testing.T) {
	s := profiledPart(t)
	if _, err := apply(t, s, "extrude", `{"sketchIndex":0,"distance":"20 mm"}`); err != nil {
		t.Fatalf("seed extrude: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	box := def.SurfaceBodies().Item(0)
	bottom := topo.DescribeFace(lowestFace(box))
	top := topo.DescribeFace(topFaceOfBody(box))
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 1, "extent": "from-to", "operation": "new",
		"fromFaceGeom": geomOf(bottom), "toFaceGeom": geomOf(top),
	})
	if _, err := apply(t, s, "extrude", string(args)); err != nil {
		t.Fatalf("from-to geom extrude: %v", err)
	}
	// Two disjoint bodies now exist (the seed + the from-to prism); the new prism spans z=0..2.
	if n := def.SurfaceBodies().Count(); n != 2 {
		t.Fatalf("expected 2 bodies (seed + from-to), got %d", n)
	}
}

// TestExtrudeFromToErrors: from-to without a fromFace, and distance-from-face without a toFace,
// are both clean argument errors.
func TestExtrudeFromToErrors(t *testing.T) {
	if _, err := apply(t, profiledPart(t), "extrude", `{"sketchIndex":0,"extent":"from-to","toFace":"origin/plane/xy"}`); err == nil {
		t.Error("from-to without a fromFace should error")
	}
	if _, err := apply(t, profiledPart(t), "extrude", `{"sketchIndex":0,"extent":"distance-from-face","distance":"5 mm"}`); err == nil {
		t.Error("distance-from-face without a toFace should error")
	}
}

// lowestFace returns the body's face with the smallest range-box centre z (the bottom cap).
func lowestFace(b *topo.Body) *topo.Face {
	var best *topo.Face
	for _, f := range b.Faces() {
		if best == nil || f.RangeBox().Center().Z < best.RangeBox().Center().Z {
			best = f
		}
	}
	return best
}

// geomOf renders a face descriptor as the {centroid, normal} selector the extent args accept.
func geomOf(g topo.GeometricFaceRef) map[string]any {
	return map[string]any{
		"centroid": []float64{float64(g.Centroid.X), float64(g.Centroid.Y), float64(g.Centroid.Z)},
		"normal":   []float64{float64(g.Normal.X), float64(g.Normal.Y), float64(g.Normal.Z)},
	}
}
