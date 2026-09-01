// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"math"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// flangedSheet builds a sheet-metal part with a base Face and one 90° flange over the wire,
// returning the router + session ready for a bends query.
func flangedSheet(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := newSheetMetalPart(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[4,3]]}`, &struct{}{})
	var face featureResult
	call(t, r, s, "features.add", `{"kind":"sheetMetalFace","args":{"sketchIndex":0}}`, &face)
	if !face.Healthy {
		t.Fatal("base Face unhealthy")
	}

	// The reference key is binary, so marshal the payload (json escapes control bytes)
	// rather than concatenating it into a JSON literal.
	payload, err := json.Marshal(map[string]any{
		"kind": "sheetMetalFlange",
		"args": map[string]any{"edge": topXEdgeKey(t, s), "height": "10 mm"},
	})
	if err != nil {
		t.Fatalf("marshal flange args: %v", err)
	}
	var flange featureResult
	call(t, r, s, "features.add", string(payload), &flange)
	if !flange.Healthy {
		t.Fatal("flange unhealthy")
	}
	return r, s
}

// topXEdgeKey returns the reference key of an X-aligned top edge of the active part's body —
// a valid edge to flange from.
func topXEdgeKey(t *testing.T, s *app.Session) string {
	t.Helper()
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	body := part.Features().Result()[0]
	maxZ := math.Inf(-1)
	for _, e := range body.Edges() {
		if z := e.StartVertex().Point().Z; z > maxZ {
			maxZ = z
		}
	}
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(a.X-b.X) > 1e-6 && math.Abs(a.Y-b.Y) < 1e-6 && math.Abs(a.Z-maxZ) < 1e-6 {
			return string(e.ReferenceKey())
		}
	}
	t.Fatal("no X-aligned top edge on the sheet")
	return ""
}

// TestSheetMetalBendsOverWire a flanged sheet reports its bend lineage: one 90° bend with a
// positive developed allowance, and the total allowance equals that bend's.
func TestSheetMetalBendsOverWire(t *testing.T) {
	t.Parallel()
	r, s := flangedSheet(t)
	var res wire.BendsResult
	call(t, r, s, wire.MethodSheetMetalBends, "{}", &res)
	if len(res.Bends) != 1 {
		t.Fatalf("bends = %d, want 1", len(res.Bends))
	}
	b := res.Bends[0]
	if math.Abs(b.Angle-90) > 1e-6 {
		t.Errorf("bend angle = %v deg, want 90", b.Angle)
	}
	if b.Allowance <= 0 || b.Radius <= 0 || b.Thickness <= 0 {
		t.Errorf("bend developed values must be positive: %+v", b)
	}
	if math.Abs(res.TotalAllowance-b.Allowance) > 1e-12 {
		t.Errorf("total allowance = %v, want %v", res.TotalAllowance, b.Allowance)
	}
}

// TestSheetMetalBendsEmptyWithoutBends a flat base sheet (no folds) reports no bends.
func TestSheetMetalBendsEmptyWithoutBends(t *testing.T) {
	t.Parallel()
	r, s := newSheetMetalPart(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[4,3]]}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"sheetMetalFace","args":{"sketchIndex":0}}`, &featureResult{})

	var res wire.BendsResult
	call(t, r, s, wire.MethodSheetMetalBends, "{}", &res)
	if len(res.Bends) != 0 || res.TotalAllowance != 0 {
		t.Errorf("flat sheet bends = %+v, want empty", res)
	}
}

// TestSheetMetalBendsRejectsPlainPart bends on an ordinary part errors (no sheet-metal rule).
func TestSheetMetalBendsRejectsPlainPart(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	if _, err := r.Handle(s, wire.MethodSheetMetalBends, []byte("{}")); err == nil {
		t.Fatal("bends on a plain part must error")
	}
}

// TestSheetMetalUnfoldOverWire a flanged sheet unfolds over the wire to a flat with a positive
// gauge/area, extents that exceed the folded footprint (the developed tab), and one fold line.
func TestSheetMetalUnfoldOverWire(t *testing.T) {
	t.Parallel()
	r, s := flangedSheet(t) // 40×30 mm base, 10 mm flange
	var res wire.UnfoldResult
	call(t, r, s, wire.MethodSheetMetalUnfold, "{}", &res)
	flat := res.Flat
	if flat.Thickness <= 0 || flat.Area <= 0 {
		t.Errorf("flat gauge/area must be positive: %+v", flat)
	}
	// Base is 4×3 cm; the flange develops a tab beyond it, so one extent exceeds 4 cm.
	if w, h := flat.Extents.Max.X-flat.Extents.Min.X, flat.Extents.Max.Y-flat.Extents.Min.Y; w < 3.9 || h < 3.9 || math.Max(w, h) < 4.1 {
		t.Errorf("flat extents = %.3f × %.3f cm, want base 4×3 plus a developed tab", w, h)
	}
	if len(flat.Bends) != 1 || math.Abs(flat.Bends[0].Angle-90) > 1e-6 {
		t.Errorf("flat fold lines = %+v, want one 90° bend", flat.Bends)
	}
}

// TestSheetMetalUnfoldRejectsPlainPart unfold on an ordinary part errors (no sheet-metal rule).
func TestSheetMetalUnfoldRejectsPlainPart(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	if _, err := r.Handle(s, wire.MethodSheetMetalUnfold, []byte("{}")); err == nil {
		t.Fatal("unfold on a plain part must error")
	}
}
