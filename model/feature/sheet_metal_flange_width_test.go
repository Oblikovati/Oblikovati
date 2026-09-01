// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
)

// Partial-width flanges (#1958). Every flange used to span its whole edge, so a bracket tab could
// not be modelled at all. The tests measure WHERE along the edge the wall stands, not just that
// one was built — a width that never reached the extruder produces a perfectly good full-width
// wall.

// wallSpanY returns the extent along the seed sheet's flanged edge (the X axis) of the material
// that stands well above the sheet — i.e. the raised wall's span.
func wallSpanX(body *topo.Body) (from, to float64) {
	from, to = stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range body.Vertices() {
		if float64(v.Point().Z) < 0.5 {
			continue // still part of the flat sheet
		}
		x := float64(v.Point().X)
		from, to = stdmath.Min(from, x), stdmath.Max(to, x)
	}
	return from, to
}

// widthFlange folds a 1 cm flange of the given width extent onto the 4 cm seed sheet.
func widthFlange(t *testing.T, w FlangeWidth) *topo.Body {
	t.Helper()
	return flangeBody(t, &SheetMetalFlangeDefinition{
		Height: constClosure(1.0), Radius: constClosure(0.3), Width: w,
	})
}

// TestFullEdgeWidthIsTheDefault: the zero value spans the whole edge, so a flange authored before
// widths existed is unchanged.
func TestFullEdgeWidthIsTheDefault(t *testing.T) {
	t.Parallel()
	from, to := wallSpanX(widthFlange(t, FlangeWidth{}))
	if stdmath.Abs(from) > 1e-6 || stdmath.Abs(to-4) > 1e-6 {
		t.Errorf("default flange wall spans [%.4f, %.4f], want the whole [0, 4] edge", from, to)
	}
}

// TestCenteredWidthCentresTheTab: a 2 cm tab on a 4 cm edge stands over [1, 3] — the case a
// bracket needs, and the one a full-width wall would silently satisfy.
func TestCenteredWidthCentresTheTab(t *testing.T) {
	t.Parallel()
	from, to := wallSpanX(widthFlange(t, FlangeWidth{Type: WidthCentered, Width: constClosure(2.0)}))
	if stdmath.Abs(from-1) > 1e-6 || stdmath.Abs(to-3) > 1e-6 {
		t.Errorf("centred 2 cm tab spans [%.4f, %.4f], want [1, 3]", from, to)
	}
}

// TestOffsetWidthsTrimBothEnds: offsets take material off each end independently, which is how a
// chassis wall leaves room for the flanges on the edges beside it.
func TestOffsetWidthsTrimBothEnds(t *testing.T) {
	t.Parallel()
	from, to := wallSpanX(widthFlange(t, FlangeWidth{
		Type: WidthOffsets, Offset: constClosure(0.5), Offset2: constClosure(1.5),
	}))
	if stdmath.Abs(from-0.5) > 1e-6 || stdmath.Abs(to-2.5) > 1e-6 {
		t.Errorf("offset wall spans [%.4f, %.4f], want [0.5, 2.5]", from, to)
	}
}

// TestOffsetAndWidthPlacesTheTab: an offset from the start plus a width puts the tab exactly where
// a hole pattern or a mating part needs it.
func TestOffsetAndWidthPlacesTheTab(t *testing.T) {
	t.Parallel()
	from, to := wallSpanX(widthFlange(t, FlangeWidth{
		Type: WidthOffsetAndWidth, Offset: constClosure(1.0), Width: constClosure(1.5),
	}))
	if stdmath.Abs(from-1) > 1e-6 || stdmath.Abs(to-2.5) > 1e-6 {
		t.Errorf("offset+width tab spans [%.4f, %.4f], want [1, 2.5]", from, to)
	}
}

// TestPartialWidthDevelopsAsAPartialTab: the flat pattern lays a bend out from the placement's
// bend LINE, so a partial wall has to report the sub-span — reporting the whole edge would unfold
// a full-width tab and cut a blank that is wrong in the one dimension the operator relies on.
func TestPartialWidthDevelopsAsAPartialTab(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
		Width: FlangeWidth{Type: WidthCentered, Width: constClosure(2.0)},
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("partial flange sick: %+v", pf.Health())
	}
	placement, ok := pf.Definition().(*SheetMetalFlangeFeature).Placement()
	if !ok {
		t.Fatal("no placement captured")
	}
	length := float64(placement.AxisStart.DistanceTo(placement.AxisEnd))
	if stdmath.Abs(length-2) > 1e-6 {
		t.Errorf("developed bend line is %.4f long, want the tab's 2 (not the edge's 4)", length)
	}
	if x := float64(placement.AxisStart.X); stdmath.Abs(x-1) > 1e-6 {
		t.Errorf("developed bend line starts at x=%.4f, want 1", x)
	}
}

// TestWidthOutsideTheEdgeIsRefused: a span that runs off the edge, or inverts, would otherwise
// build a wall of a different width than the one asked for and say nothing.
func TestWidthOutsideTheEdgeIsRefused(t *testing.T) {
	t.Parallel()
	for name, w := range map[string]FlangeWidth{
		"wider than the edge":  {Type: WidthCentered, Width: constClosure(6)},
		"zero width":           {Type: WidthCentered, Width: constClosure(0)},
		"offsets that cross":   {Type: WidthOffsets, Offset: constClosure(3), Offset2: constClosure(2)},
		"tab past the far end": {Type: WidthOffsetAndWidth, Offset: constClosure(3), Width: constClosure(2)},
	} {
		t.Run(name, func(t *testing.T) {
			pf, _ := flangeFeature(t, &SheetMetalFlangeDefinition{
				Height: constClosure(1.0), Radius: constClosure(0.3), Width: w,
			})
			if pf.Health().OK() {
				t.Errorf("a width %s should be refused, not clamped", name)
			}
		})
	}
}

// TestContourFlangeTakesAWidthToo: the swept flange spans its edge the same way, and #1958 covers
// both — a contour flange that ignored the width would be the one full-width wall left.
func TestContourFlangeTakesAWidthToo(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalContourFlangeFeatures(fs).Add(&SheetMetalContourFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Profile: lProfile(),
		Width: FlangeWidth{Type: WidthCentered, Width: constClosure(2.0)},
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("partial contour flange sick: %+v", pf.Health())
	}
	from, to := wallSpanX(fs.Result()[0])
	if stdmath.Abs(from-1) > 1e-6 || stdmath.Abs(to-3) > 1e-6 {
		t.Errorf("centred contour flange spans [%.4f, %.4f], want [1, 3]", from, to)
	}
}

// TestFlangeWidthRoundTrips: the width decides a dimension of the part, so losing it on reopen
// would rebuild a full-width wall from the same recipe.
func TestFlangeWidthRoundTrips(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: []byte("edge"), Height: constClosure(1.0),
		Width: FlangeWidth{Type: WidthOffsetAndWidth, Offset: constClosure(1.0), Width: constClosure(1.5)},
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalFlange.Width
	if d == nil || d.Type != "offsetWidth" || d.Offset != 1.0 || d.Width != 1.5 {
		t.Fatalf("serialized width = %+v, want the offsetWidth extent", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	w := fresh.Item(0).Definition().(*SheetMetalFlangeFeature).Definition().Width
	if w.Type != WidthOffsetAndWidth || evalFloat(w.Offset) != 1.0 || evalFloat(w.Width) != 1.5 {
		t.Errorf("restored width = %+v, want the offsetWidth extent at 1.0/1.5", w)
	}
}

// TestFullEdgeWidthSerializesNothing: the default must leave the recipe exactly as it was before
// widths existed.
func TestFullEdgeWidthSerializesNothing(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: []byte("edge"), Height: constClosure(1.0),
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if w := data[0].SheetMetalFlange.Width; w != nil {
		t.Errorf("a full-edge flange serialized a width block: %+v", w)
	}
}

// TestUnknownWidthExtentIsRefused: "fromTo" is Inventor's fifth extent and is not offered, so it
// must be refused rather than resolving to the full edge.
func TestUnknownWidthExtentIsRefused(t *testing.T) {
	t.Parallel()
	if _, ok := ParseWidthExtent("fromTo"); ok {
		t.Error(`ParseWidthExtent("fromTo") should not resolve — it needs entity references`)
	}
	if w, ok := ParseWidthExtent(""); !ok || w != WidthFullEdge {
		t.Errorf("the empty extent should be the full edge, got %v/%v", w, ok)
	}
}
