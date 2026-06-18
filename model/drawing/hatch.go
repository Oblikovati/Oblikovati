// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
)

// Hatch regions (M14-F08, #638): fill a rectangular region of a drawing sketch with a hatch pattern
// (a family of parallel lines at an angle and spacing; cross-hatch adds the perpendicular family).
// The fill lines are generated as drawing curves, clipped to the rectangle, and re-derive with the
// sketch.

// hatchRegion is a rectangular fill on a sketch: its rectangle (sheet mm), pattern, and line spacing.
type hatchRegion struct {
	x, y, w, h float64
	pattern    types.HatchPattern
	spacing    float64
}

// hatchPatternSpec is a built-in pattern's default geometry.
type hatchPatternSpec struct {
	angleDeg float64
	spacing  float64
	double   bool // cross-hatch: add the perpendicular family
}

// hatchCatalog maps each built-in pattern to its line family.
var hatchCatalog = map[types.HatchPattern]hatchPatternSpec{
	types.HatchGeneral: {angleDeg: 45, spacing: 3, double: false},
	types.HatchANSI31:  {angleDeg: 45, spacing: 2, double: false},
	types.HatchCross:   {angleDeg: 45, spacing: 3, double: true},
}

// AddHatchRegion fills the rectangle (x, y, w, h) on the named sketch (created if sketchName is
// blank) with the given pattern; scale overrides the line spacing (0 ⇒ the pattern default). It
// errors on a non-positive size or an unknown sketch.
func (ss *DrawingSketches) AddHatchRegion(sketchName string, x, y, w, h float64, pattern types.HatchPattern, scale float64) (*DrawingSketch, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("drawing: a hatch region needs a positive size, got %g×%g", w, h)
	}
	s := ss.resolveOrCreate(sketchName)
	if s == nil {
		return nil, fmt.Errorf("drawing: no sketch %q to add a hatch region to", sketchName)
	}
	spacing := scale
	if spacing <= 0 {
		spacing = hatchCatalog[pattern].spacing
	}
	s.hatches = append(s.hatches, hatchRegion{x: x, y: y, w: w, h: h, pattern: pattern, spacing: spacing})
	s.rebuild()
	return s, nil
}

// resolveOrCreate returns the named sketch, creating one when the name is blank.
func (ss *DrawingSketches) resolveOrCreate(name string) *DrawingSketch {
	if name == "" {
		return ss.Add("")
	}
	s, _ := ss.ByName(name)
	return s
}

// hatchRegionCurves generates a hatch region's fill lines (clipped to its rectangle).
func hatchRegionCurves(r hatchRegion) []DrawingCurve {
	spec := hatchCatalog[r.pattern]
	out := hatchFamily(r, spec.angleDeg, r.spacing)
	if spec.double {
		out = append(out, hatchFamily(r, spec.angleDeg+90, r.spacing)...)
	}
	return out
}

// hatchFamily generates one family of parallel hatch lines at angleDeg, spaced by spacing, each
// clipped to the region's rectangle. Lines are anchored at the rectangle centre and stepped along
// the perpendicular, so the family always crosses the rectangle regardless of its sheet position.
func hatchFamily(r hatchRegion, angleDeg, spacing float64) []DrawingCurve {
	a := angleDeg * math.Pi / 180
	dx, dy := math.Cos(a), math.Sin(a) // line direction
	nx, ny := -dy, dx                  // perpendicular (offset axis)
	cx, cy := r.x+r.w/2, r.y+r.h/2     // rectangle centre (the offset origin)
	lo, hi := offsetRange(r, cx, cy, nx, ny)
	span := r.w + r.h // long enough to cross the rectangle from its centreline
	var out []DrawingCurve
	for off := math.Ceil(lo/spacing) * spacing; off <= hi; off += spacing {
		bx, by := cx+nx*off, cy+ny*off
		if x0, y0, x1, y1, ok := clipLineToRect(bx-dx*span, by-dy*span, bx+dx*span, by+dy*span, r); ok {
			out = append(out, dimSegment(x0, y0, x1, y1))
		}
	}
	return out
}

// offsetRange returns the min/max projection of the rectangle's corners onto the (nx, ny) axis,
// measured from the rectangle centre (cx, cy).
func offsetRange(r hatchRegion, cx, cy, nx, ny float64) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, c := range [4][2]float64{{r.x, r.y}, {r.x + r.w, r.y}, {r.x, r.y + r.h}, {r.x + r.w, r.y + r.h}} {
		p := (c[0]-cx)*nx + (c[1]-cy)*ny
		lo, hi = math.Min(lo, p), math.Max(hi, p)
	}
	return lo, hi
}

// clipLineToRect clips the segment (x0,y0)-(x1,y1) to the rectangle via Liang–Barsky, returning the
// clipped endpoints and whether any part lies inside.
func clipLineToRect(x0, y0, x1, y1 float64, r hatchRegion) (float64, float64, float64, float64, bool) {
	dx, dy := x1-x0, y1-y0
	t0, t1 := 0.0, 1.0
	edges := [4][2]float64{{-dx, x0 - r.x}, {dx, r.x + r.w - x0}, {-dy, y0 - r.y}, {dy, r.y + r.h - y0}}
	for _, e := range edges {
		p, q := e[0], e[1]
		if p == 0 {
			if q < 0 {
				return 0, 0, 0, 0, false // parallel and outside
			}
			continue
		}
		t := q / p
		if p < 0 {
			t0 = math.Max(t0, t)
		} else {
			t1 = math.Min(t1, t)
		}
	}
	if t0 > t1 {
		return 0, 0, 0, 0, false
	}
	return x0 + t0*dx, y0 + t0*dy, x0 + t1*dx, y0 + t1*dy, true
}
