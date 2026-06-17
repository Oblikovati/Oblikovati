// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// Flat-pattern DXF export (M13-F04, #378): a developed sheet-metal flat pattern written to DXF
// with its outline, bend lines and punch tokens on separate named layers, so a CAM/laser/punch
// programmer can pick the manufacturing geometry off the layers it expects. It reuses the DXF
// codec (kernel/exchange/dxf) — only the flat → neutral-drawing conversion is new here.

// FlatExportLayers names the DXF layers the flat pattern's geometry classes land on. A zero
// value (empty names) resolves to the defaults via withDefaults.
type FlatExportLayers struct {
	Outline  string // the flat footprint boundary
	BendUp   string // fold-up bend lines
	BendDown string // fold-down bend lines
	Punches  string // punch outlines + tokens
}

// DefaultFlatExportLayers is the layer scheme used when the caller passes none.
func DefaultFlatExportLayers() FlatExportLayers {
	return FlatExportLayers{Outline: "Outline", BendUp: "Bend-Up", BendDown: "Bend-Down", Punches: "Punches"}
}

// withDefaults fills any empty layer name from the default scheme.
func (l FlatExportLayers) withDefaults() FlatExportLayers {
	d := DefaultFlatExportLayers()
	if l.Outline == "" {
		l.Outline = d.Outline
	}
	if l.BendUp == "" {
		l.BendUp = d.BendUp
	}
	if l.BendDown == "" {
		l.BendDown = d.BendDown
	}
	if l.Punches == "" {
		l.Punches = d.Punches
	}
	return l
}

// FlatPatternToDrawing converts a developed flat pattern into the neutral drawing model: the
// footprint outline on the Outline layer and every fold line on the Bend-Up/Bend-Down layer
// per its direction. Punch tokens are added by the caller from the flat's punch representation.
func FlatPatternToDrawing(flat *feature.FlatPattern, layers FlatExportLayers) []drawing.Entity {
	layers = layers.withDefaults()
	out := flatOutlineEntities(flat, layers.Outline)
	for _, b := range flat.Bends {
		layer := layers.BendUp
		if b.FoldDown {
			layer = layers.BendDown
		}
		out = append(out, &drawing.Line{Layer: layer, Start: point2to3(b.A), End: point2to3(b.B)})
	}
	return append(out, flatPunchEntities(flat, layers.Punches)...)
}

// flatOutlineEntities emits the flat footprint as line segments on the outline layer: the body's
// edges that lie on the top plane and bound exactly one in-plane (top) face — i.e. the boundary
// between a flat top face and a vertical wall. A seam between two coplanar top faces (where a tab
// meets the base) bounds two top faces and is skipped, so only the silhouette is drawn.
func flatOutlineEntities(flat *feature.FlatPattern, layer string) []drawing.Entity {
	plane := flat.Plane
	n := plane.Normal().AsVector()
	origin := plane.Origin()
	var out []drawing.Entity
	for _, e := range flat.Body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if !onPlaneAt(a, origin, n, flat.Thickness) || !onPlaneAt(b, origin, n, flat.Thickness) {
			continue
		}
		if inPlaneFaceCount(e, n) != 1 {
			continue
		}
		out = append(out, &drawing.Line{
			Layer: layer,
			Start: point2to3(plane.ToSketch(a)),
			End:   point2to3(plane.ToSketch(b)),
		})
	}
	return out
}

// flatPunchEntities emits each punch representation as its closed outline plus a TEXT token at
// the centroid, both on the punch layer — the manufacturing tag a CAM programmer reads.
func flatPunchEntities(flat *feature.FlatPattern, layer string) []drawing.Entity {
	var out []drawing.Entity
	for _, p := range flat.Punches {
		if len(p.Outline) < 2 {
			continue
		}
		pts := make([][2]float64, len(p.Outline))
		for i, q := range p.Outline {
			pts[i] = [2]float64{float64(q.X), float64(q.Y)}
		}
		out = append(out, &drawing.LwPolyline{Layer: layer, Closed: true, Points: pts})
		if p.Token != "" {
			c, h := punchLabel(p.Outline)
			out = append(out, &drawing.Text{Layer: layer, Position: [3]float64{c[0], c[1], 0}, Height: h, Value: p.Token})
		}
	}
	return out
}

// punchLabel returns where to place a punch's token (the outline centroid) and a legible text
// height (a fraction of the smaller bounding-box dimension, with a small floor).
func punchLabel(outline []math.Point2) (centroid [2]float64, height float64) {
	minX, minY := stdmath.Inf(1), stdmath.Inf(1)
	maxX, maxY := stdmath.Inf(-1), stdmath.Inf(-1)
	var sx, sy float64
	for _, p := range outline {
		x, y := float64(p.X), float64(p.Y)
		sx, sy = sx+x, sy+y
		minX, minY = stdmath.Min(minX, x), stdmath.Min(minY, y)
		maxX, maxY = stdmath.Max(maxX, x), stdmath.Max(maxY, y)
	}
	n := float64(len(outline))
	span := stdmath.Min(maxX-minX, maxY-minY)
	return [2]float64{sx / n, sy / n}, stdmath.Max(0.3*span, flatTokenMinHeight)
}

// onPlaneAt reports whether p sits on the plane offset height h along normal n from origin
// (the flat body's top face is at h = thickness; the bottom at h = 0).
func onPlaneAt(p, origin math.Point3, n math.Vector3, h float64) bool {
	return stdmath.Abs(origin.VectorTo(p).Dot(n)-h) < flatPlaneTol
}

// inPlaneFaceCount counts an edge's adjacent faces whose normal is parallel to n (a flat,
// in-plane face). An outline edge has exactly one; an interior seam between two top faces has two.
func inPlaneFaceCount(e *topo.Edge, n math.Vector3) int {
	count := 0
	for _, f := range e.Faces() {
		fn := f.Geometry().NormalAt(0, 0)
		l := float64(fn.Length())
		if l == 0 {
			continue
		}
		if stdmath.Abs(fn.Scale(1/l).Dot(n)) > flatParallelTol {
			count++
		}
	}
	return count
}

// point2to3 lifts a 2D flat-plane point into a Z=0 drawing coordinate.
func point2to3(p math.Point2) [3]float64 { return [3]float64{float64(p.X), float64(p.Y), 0} }

const (
	// flatPlaneTol bounds "lies on the top/bottom plane" (database units, cm).
	flatPlaneTol = 1e-6
	// flatParallelTol is the |normal·planeNormal| above which a face counts as in-plane (flat).
	flatParallelTol = 0.999
	// flatTokenMinHeight floors a punch token's text height (database units, cm).
	flatTokenMinHeight = 0.1
)

// ExportFlatPatternDXF encodes a developed flat pattern as an ASCII DXF of the given version,
// returning the bytes and the entity count. Layers default when layers is the zero value.
//
//	data, n, err := exchange.ExportFlatPatternDXF(flat, exchange.FlatExportLayers{}, types.DXFR2018)
func ExportFlatPatternDXF(flat *feature.FlatPattern, layers FlatExportLayers, version types.DXFVersion) ([]byte, int, error) {
	return encodeDXF(FlatPatternToDrawing(flat, layers), version)
}

// ExportFlatPatternDXFFile writes ExportFlatPatternDXF's output to path, returning the entity count.
func ExportFlatPatternDXFFile(flat *feature.FlatPattern, path string, layers FlatExportLayers, version types.DXFVersion) (int, error) {
	return writeDXFFile(FlatPatternToDrawing(flat, layers), path, version)
}
