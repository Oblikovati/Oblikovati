// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Drawing dimensions — the LINEAR family (M48 #2225 split of dimension.go). A linear dimension
// annotates the distance between two model vertices (by reference key) on a base view — aligned,
// horizontal or vertical — plus the baseline/chain sets built from them. It re-resolves the two
// vertices on recompute (associativity) and rebuilds the extension-line + dimension-line glyph.

// AddLinear adds a linear dimension on the named base view between two pick points (sheet mm),
// each snapped to the nearest projected model vertex so the value tracks the model. dimType
// selects the measured component; offset stands the dimension line off the points (sheet mm).
func (ds *DrawingDimensions) AddLinear(name, viewName string, dimType types.DrawingDimensionType, x1, y1, x2, y2, offset float64) (*DrawingDimension, error) {
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	keyA, okA := nearestVertexKey(body, view, basis, x1, y1)
	keyB, okB := nearestVertexKey(body, view, basis, x2, y2)
	if !okA || !okB {
		return nil, fmt.Errorf(errViewNoVertices, viewName)
	}
	return ds.addLinearFromKeys(name, viewName, dimType, keyA, keyB, offset), nil
}

// addLinearFromKeys creates a linear dimension between two already-resolved vertex keys.
func (ds *DrawingDimensions) addLinearFromKeys(name, viewName string, dimType types.DrawingDimensionType, keyA, keyB []byte, offset float64) *DrawingDimension {
	d := &DrawingDimension{name: ds.uniqueName(name), dimType: dimType, viewName: viewName, keyA: keyA, keyB: keyB, offset: offset}
	ds.recompute(d)
	ds.items = append(ds.items, d)
	return d
}

// AddBaselineSet adds a baseline set: a linear dimension from the first pick point to each of the
// others, stacked at increasing offsets (the datum-referenced common case). Each point is snapped
// to the nearest projected model vertex, so every dimension stays associative.
func (ds *DrawingDimensions) AddBaselineSet(viewName string, dimType types.DrawingDimensionType, points [][2]float64) ([]*DrawingDimension, error) {
	keys, err := ds.snapPoints(viewName, points)
	if err != nil {
		return nil, err
	}
	const baseGap = 14.0
	out := make([]*DrawingDimension, 0, len(keys)-1)
	for i := 1; i < len(keys); i++ {
		out = append(out, ds.addLinearFromKeys("", viewName, dimType, keys[0], keys[i], -baseGap*float64(i)))
	}
	return out, nil
}

// AddChainSet adds a chain set: a linear dimension between each consecutive pair of pick points,
// all on one line (running dimensions). Each point is snapped to the nearest model vertex.
func (ds *DrawingDimensions) AddChainSet(viewName string, dimType types.DrawingDimensionType, points [][2]float64) ([]*DrawingDimension, error) {
	keys, err := ds.snapPoints(viewName, points)
	if err != nil {
		return nil, err
	}
	const chainOffset = -14.0
	out := make([]*DrawingDimension, 0, len(keys)-1)
	for i := 0; i+1 < len(keys); i++ {
		out = append(out, ds.addLinearFromKeys("", viewName, dimType, keys[i], keys[i+1], chainOffset))
	}
	return out, nil
}

// AddSetForViewCorners dimensions a base view's bounding-box corners as a baseline or chain set —
// the single-action dimension set (a full multi-pick set is a follow-up). Aligned measurement
// gives non-degenerate values (width / diagonal / height).
func (ds *DrawingDimensions) AddSetForViewCorners(viewName string, dimType types.DrawingDimensionType, baseline bool) ([]*DrawingDimension, error) {
	view, ok := ds.views.ByName(viewName)
	if !ok {
		return nil, fmt.Errorf("drawing: no view %q to dimension", viewName)
	}
	minX, minY, maxX, maxY, ok := view.BoundsMM()
	if !ok {
		return nil, fmt.Errorf("drawing: view %q has no geometry to dimension", viewName)
	}
	pts := [][2]float64{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
	if baseline {
		return ds.AddBaselineSet(viewName, dimType, pts)
	}
	return ds.AddChainSet(viewName, dimType, pts)
}

// snapPoints resolves each pick point to a model-vertex key on the view; it needs at least two
// points and a resolvable model.
func (ds *DrawingDimensions) snapPoints(viewName string, points [][2]float64) ([][]byte, error) {
	if len(points) < 2 {
		return nil, fmt.Errorf("drawing: a dimension set needs at least two points, got %d", len(points))
	}
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	keys := make([][]byte, len(points))
	for i, p := range points {
		k, ok := nearestVertexKey(body, view, basis, p[0], p[1])
		if !ok {
			return nil, fmt.Errorf(errViewNoVertices, viewName)
		}
		keys[i] = k
	}
	return keys, nil
}

// recomputeLinear re-binds the two vertices, re-projects them and rebuilds the linear glyph.
func (ds *DrawingDimensions) recomputeLinear(d *DrawingDimension, view *DrawingView, body *topo.Body, basis hlr.View) {
	va, okA := body.FindVertexByKey(d.keyA)
	vb, okB := body.FindVertexByKey(d.keyB)
	if !okA || !okB {
		return
	}
	p1 := hlr.ProjectPoint(basis, va.Point())
	p2 := hlr.ProjectPoint(basis, vb.Point())
	ds.buildLinearGlyph(d, view, p1, p2, measureMM(d.dimType, p1, p2))
}

// buildLinearGlyph measures the (already projected) endpoints, decorates the value and rebuilds the
// linear dimension glyph and text anchor — the shared tail of a picked linear dimension and a
// retrieved model dimension (#1991).
func (ds *DrawingDimensions) buildLinearGlyph(d *DrawingDimension, view *DrawingView, p1, p2 gmath.Point2, valueMM float64) {
	d.valueMM = valueMM
	d.text = d.decorate(formatDimValue(valueMM, ds.decimals()))
	s1, s2 := view.place(p1), view.place(p2)
	ax, ay := dimensionAxis(d.dimType, s1, s2)
	var mx, my float64
	d.curves, mx, my = linearDimensionCurves(s1, s2, ax, ay, d.offset)
	// Lift the text off the dimension line, on the side away from the measured geometry.
	d.setTextAnchor(mx, my, -ay, ax, sign(d.offset))
}

// linearDimensionCurves builds a linear dimension's glyph (sheet mm): an extension line from each
// measured point out to the dimension line, the dimension line itself, and an arrowhead at each
// end. The dimension line is parallel to (ax, ay) and offset along its perpendicular by offset.
// It also returns the dimension line's midpoint (the text anchor).
func linearDimensionCurves(s1, s2 gmath.Point2, ax, ay, offset float64) (curves []DrawingCurve, anchorX, anchorY float64) {
	nx, ny := -ay, ax // perpendicular to the measurement axis
	s1x, s1y := float64(s1.X), float64(s1.Y)
	s2x, s2y := float64(s2.X), float64(s2.Y)
	lvl := s1x*nx + s1y*ny + offset // n-coordinate of the dimension line
	e1x, e1y := s1x+offset*nx, s1y+offset*ny
	d2 := lvl - (s2x*nx + s2y*ny)
	e2x, e2y := s2x+d2*nx, s2y+d2*ny
	out := []DrawingCurve{
		dimSegment(s1x, s1y, e1x, e1y),
		dimSegment(s2x, s2y, e2x, e2y),
		dimSegment(e1x, e1y, e2x, e2y),
	}
	out = append(out, arrowheadCurves(e1x, e1y, e2x, e2y)...)
	out = append(out, arrowheadCurves(e2x, e2y, e1x, e1y)...)
	return out, (e1x + e2x) / 2, (e1y + e2y) / 2
}
