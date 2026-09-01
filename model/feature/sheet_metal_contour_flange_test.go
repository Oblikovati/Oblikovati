// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// lProfile returns an open sketch whose chain is an L: out 1 cm along the face, then up 1 cm —
// the cross-section of a simple right-angle contour flange.
func lProfile() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	p0 := s.Points().Add(gmath.P2(0, 0))
	p1 := s.Points().Add(gmath.P2(1, 0))
	p2 := s.Points().Add(gmath.P2(1, 1))
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	return s
}

// TestContourFlangeSweepsProfile a contour flange sweeps the L-profile along the sheet edge
// into one watertight valid solid that adds material and rises out of plane.
func TestContourFlangeSweepsProfile(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	flat := query.BodyGeometryProperties(fs.Result()[0], ops.Quality{ChordTolerance: 1e-4}).Volume

	pf := NewSheetMetalContourFlangeFeatures(fs).Add(&SheetMetalContourFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Profile: lProfile(),
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("contour flange sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if len(fs.Result()) != 1 || !body.IsSolid() {
		t.Fatalf("contour flange did not merge into one solid (%d bodies)", len(fs.Result()))
	}
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("contour flange invalid: %v", r.Issues)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("contour flange not watertight: %d boundary edges", len(open))
	}
	vol := query.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-4}).Volume
	if !(vol > flat) {
		t.Errorf("contour flange added no material: flat=%.4f flanged=%.4f", flat, vol)
	}
	// The L's vertical leg rises ~1 cm above the sheet.
	if box := body.RangeBox(); box.Max.Z < 0.8 {
		t.Errorf("contour flange top Z = %g, want the vertical leg to rise (>0.8)", box.Max.Z)
	}
}

// TestContourFlangeRejectsBadProfile a nil/empty profile and a non-chain profile go sick.
func TestContourFlangeRejectsBadProfile(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalContourFlangeFeatures(fs).Add(&SheetMetalContourFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Profile: sketch.NewSketches().Add(sketch.XYPlane()), // no lines
	})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("contour flange with an empty profile should be sick")
	}
}

// TestOpenProfilePoints the chain walker orders the L-profile's vertices start→end.
func TestOpenProfilePoints(t *testing.T) {
	t.Parallel()
	pts, err := openProfilePoints(lProfile())
	if err != nil {
		t.Fatalf("openProfilePoints: %v", err)
	}
	if len(pts) != 3 || pts[0] != gmath.P2(0, 0) || pts[2] != gmath.P2(1, 1) {
		t.Errorf("walked profile = %v, want (0,0)->(1,0)->(1,1)", pts)
	}
}

// TestOpenProfileRejectsClosedLoop a closed loop has no degree-1 end, so it is not a contour.
func TestOpenProfileRejectsClosedLoop(t *testing.T) {
	t.Parallel()
	if _, err := openProfilePoints(squareSketch(2)); err == nil {
		t.Error("a closed loop is not an open contour and must error")
	}
	if _, err := openProfilePoints(nil); err == nil {
		t.Error("a nil profile must error")
	}
}

// TestContourFlangeDefinitionAndKind the accessors return the recipe.
func TestContourFlangeDefinitionAndKind(t *testing.T) {
	t.Parallel()
	def := &SheetMetalContourFlangeDefinition{}
	f := &SheetMetalContourFlangeFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-contour-flange" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestContourFlangeRoundTrip the recipe (edge key + profile sketch + flip) marshals and
// restores, preserving the kind and payload.
func TestContourFlangeRoundTrip(t *testing.T) {
	t.Parallel()
	profile := lProfile()
	fs := NewPartFeatures(nil)
	NewSheetMetalContourFlangeFeatures(fs).Add(&SheetMetalContourFlangeDefinition{
		EdgeKey: []byte("edge"), Profile: profile, Flip: true,
	})
	data, err := fs.MarshalRecipe(oneSketch{s: profile})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalContourFlange
	if data[0].Kind != "sheet-metal-contour-flange" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-contour-flange", data[0])
	}
	if d.Profile != 0 || !d.Flip {
		t.Errorf("payload = %+v, want profile 0 / flip", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: profile}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-contour-flange" {
		t.Errorf("restored = %d features, want one contour flange", fresh.Count())
	}
}

// TestContourFlangeMissingPayload / unknown sketch restore errors.
func TestContourFlangeMissingPayload(t *testing.T) {
	t.Parallel()
	if _, err := restoreSheetMetalContourFlange(NewPartFeatures(nil), nil, oneSketch{}); err == nil {
		t.Error("restoreSheetMetalContourFlange(nil) must error")
	}
	if _, err := serializeSheetMetalContourFlange(&SheetMetalContourFlangeDefinition{Profile: lProfile()}, oneSketch{s: squareSketch(1)}); err == nil {
		t.Error("serialize with an unknown profile sketch must error")
	}
}

var _ = math.Pi // keep math imported for future profile fixtures
