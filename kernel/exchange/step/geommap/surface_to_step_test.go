// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"testing"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// emitAndReparse emits via fn into a fresh writer, parses the result, and returns
// the entity for the emitted id so a test can assert its keyword/params.
func emitAndReparse(t *testing.T, fn func(*Emitter) int) (*part21.RawEntity, *part21.EntityGraph) {
	t.Helper()
	w := part21.NewWriter()
	e := NewEmitter(w, 1.0)
	id := fn(e)
	w.Add("DUMMY") // ensure DATA is non-empty regardless
	f, err := part21.Parse(w.Emit(part21.Header{SchemaIdentifiers: []string{"CONFIG_CONTROL_DESIGN"}}))
	if err != nil {
		t.Fatalf("re-parse emitted: %v", err)
	}
	ent, err := f.Graph.Lookup(id)
	if err != nil {
		t.Fatalf("lookup emitted #%d: %v", id, err)
	}
	return ent, f.Graph
}

func TestPlaneRoundTripsThroughStep(t *testing.T) {
	pl, _ := geom.NewPlane(math.P3(1, 2, 3), math.V3(0, 0, 1))
	ent, g := emitAndReparse(t, func(e *Emitter) int { return e.planeToStep(pl) })
	if ent.Keyword != "PLANE" {
		t.Fatalf("emitted keyword = %q, want PLANE", ent.Keyword)
	}
	placeRef, _ := ent.Params[1].AsRef()
	f, err := Placement(g, placeRef, 1.0)
	if err != nil {
		t.Fatalf("re-read placement: %v", err)
	}
	if !f.Origin.IsEqualTo(math.P3(1, 2, 3), 1e-9) {
		t.Errorf("plane origin round-trip = %v, want (1,2,3)", f.Origin)
	}
}

func TestCylinderRadiusRoundTrips(t *testing.T) {
	c, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 4.5)
	ent, g := emitAndReparse(t, func(e *Emitter) int { return e.cylinderToStep(c) })
	if ent.Keyword != "CYLINDRICAL_SURFACE" {
		t.Fatalf("emitted keyword = %q, want CYLINDRICAL_SURFACE", ent.Keyword)
	}
	s, err := Surface(g, ent.ID, 1.0)
	if err != nil {
		t.Fatalf("re-read surface: %v", err)
	}
	if cyl, ok := s.(geom.Cylinder); !ok || cyl.Radius != 4.5 {
		t.Errorf("cylinder radius round-trip = %v, want 4.5", s)
	}
}

func TestExportUnsupportedSurfaceErrors(t *testing.T) {
	w := part21.NewWriter()
	e := NewEmitter(w, 1.0)
	bs, _ := geom.NewBSplineSurface(1, 1,
		[][]math.Point3{{math.P3(0, 0, 0), math.P3(1, 0, 0)}, {math.P3(0, 1, 0), math.P3(1, 1, 0)}},
		[][]float64{{1, 1}, {1, 1}}, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1})
	if _, err := e.SurfaceToStep(bs); err == nil {
		t.Error("exporting a B-spline surface should error (PBI-E), not silently succeed")
	}
}
