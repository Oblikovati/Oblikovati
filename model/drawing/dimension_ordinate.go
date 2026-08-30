// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Drawing dimensions — the ORDINATE family (M48 #2225 split of dimension.go). An ordinate dimension
// measures a point's view-X or view-Y offset from a shared datum vertex, drawn as a leader to a
// common spine (so a set lines up) with no dimension line or arrowhead. Datum and point attach by
// reference key and re-resolve on recompute (associativity).

// AddOrdinateSet adds one ordinate dimension per point, each measuring the point's offset from the
// shared datum along one axis (axisHorizontal = view-X, else view-Y), drawn as a leader to its
// value with no dimension line. The datum and every point snap to the nearest projected model
// vertex, so every ordinate stays associative.
func (ds *DrawingDimensions) AddOrdinateSet(viewName string, axisHorizontal bool, datum [2]float64, points [][2]float64) ([]*DrawingDimension, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("drawing: an ordinate set needs at least one point, got 0")
	}
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	datumKey, ok := nearestVertexKey(body, view, basis, datum[0], datum[1])
	if !ok {
		return nil, fmt.Errorf(errViewNoVertices, viewName)
	}
	out := make([]*DrawingDimension, 0, len(points))
	for _, p := range points {
		key, ok := nearestVertexKey(body, view, basis, p[0], p[1])
		if !ok {
			return nil, fmt.Errorf(errViewNoVertices, viewName)
		}
		out = append(out, ds.addOrdinateFromKeys(viewName, axisHorizontal, datumKey, key))
	}
	return out, nil
}

// addOrdinateFromKeys creates one ordinate dimension between an already-resolved datum and point.
func (ds *DrawingDimensions) addOrdinateFromKeys(viewName string, axisHorizontal bool, datumKey, pointKey []byte) *DrawingDimension {
	d := &DrawingDimension{name: ds.uniqueName(""), dimType: types.OrdinateDimension, viewName: viewName, keyA: datumKey, keyB: pointKey, axisHorizontal: axisHorizontal}
	ds.recompute(d)
	ds.items = append(ds.items, d)
	return d
}

// recomputeOrdinate re-binds the datum (keyA) and measured point (keyB), measures the point's
// view-X or view-Y offset from the datum (the running coordinate), then builds the leader glyph.
func (ds *DrawingDimensions) recomputeOrdinate(d *DrawingDimension, view *DrawingView, body *topo.Body, basis hlr.View) {
	datum, okA := body.FindVertexByKey(d.keyA)
	point, okB := body.FindVertexByKey(d.keyB)
	if !okA || !okB {
		return
	}
	pd := hlr.ProjectPoint(basis, datum.Point())
	pp := hlr.ProjectPoint(basis, point.Point())
	if d.axisHorizontal {
		d.valueMM = math.Abs(float64(pp.X-pd.X)) * cmToMM
	} else {
		d.valueMM = math.Abs(float64(pp.Y-pd.Y)) * cmToMM
	}
	d.text = d.decorate(formatDimValue(d.valueMM, ds.decimals()))
	curves, mx, my, nx, ny := ordinateDimensionCurves(view, view.place(pp), d.axisHorizontal)
	d.curves = curves
	d.setTextAnchor(mx, my, nx, ny, 1)
}

// ordinateDimensionCurves builds an ordinate leader (sheet mm): a witness line from the measured
// point out to a common spine — below the view for a horizontal (view-X) ordinate, to its right
// for a vertical (view-Y) ordinate — so every ordinate text in a set lines up. It returns the text
// anchor at the spine end and the outward lift direction. There is no dimension line or arrowhead.
func ordinateDimensionCurves(view *DrawingView, point gmath.Point2, axisHorizontal bool) (curves []DrawingCurve, anchorX, anchorY, nx, ny float64) {
	px, py := float64(point.X), float64(point.Y)
	const spineGap = 14.0
	_, minY, maxX, _, ok := view.BoundsMM()
	if !ok {
		minY, maxX = py, px
	}
	if axisHorizontal {
		spineY := minY - spineGap
		return []DrawingCurve{dimSegment(px, py, px, spineY)}, px, spineY, 0, -1
	}
	spineX := maxX + spineGap
	return []DrawingCurve{dimSegment(px, py, spineX, py)}, spineX, py, 1, 0
}
