// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange/dwg"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// xyPlane is the part's XY plane, the default target for these conversion tests.
func xyPlane(t *testing.T) sketch.Plane {
	t.Helper()
	p, err := sketch.NewPlane(gmath.P3(0, 0, 0), gmath.V3(1, 0, 0).AsUnit(), gmath.V3(0, 1, 0).AsUnit())
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return p
}

// TestAdd2DEntitiesMapsEachType converts one of each entity type onto a 2D sketch
// and checks the resulting per-collection counts, including a polyline that splits
// into a line and a bulge arc.
func TestAdd2DEntitiesMapsEachType(t *testing.T) {
	part := compdef.NewPartComponentDefinition()
	sk := part.Sketches().Add(xyPlane(t))

	entities := []dwg.Entity{
		&dwg.Line{Start: [3]float64{0, 0, 0}, End: [3]float64{1, 0, 0}},
		&dwg.Circle{Center: [3]float64{0, 0, 0}, Radius: 2},
		&dwg.Arc{Center: [3]float64{0, 0, 0}, Radius: 1, StartAngle: 0, EndAngle: 1.5},
		&dwg.Point{Position: [3]float64{3, 3, 0}},
		&dwg.Ellipse{Center: [3]float64{0, 0, 0}, MajorAxis: [3]float64{4, 0, 0}, AxisRatio: 0.5},
		&dwg.Spline{ControlPoints: [][3]float64{{0, 0, 0}, {1, 1, 0}, {2, 0, 0}}, Degree: 2},
		// open polyline: (0,0)->(1,0) straight, (1,0)->(1,1) bulged -> one line + one arc
		&dwg.LwPolyline{Points: [][2]float64{{0, 0}, {1, 0}, {1, 1}}, Bulges: []float64{0, 0.5, 0}},
	}
	added, warns := add2DEntities(sk, entities)
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	if added != len(entities) {
		t.Fatalf("added %d entities, want %d", added, len(entities))
	}
	checkCount(t, "lines", sk.Lines().Count(), 2) // line entity + polyline straight segment
	checkCount(t, "circles", sk.Circles().Count(), 1)
	checkCount(t, "arcs", sk.Arcs().Count(), 2) // arc entity + polyline bulge segment
	checkCount(t, "points", sk.Points().Count(), 1)
	checkCount(t, "ellipses", sk.Ellipses().Count(), 1)
	checkCount(t, "splines", sk.Splines().Count(), 1)
}

func checkCount(t *testing.T, what string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", what, got, want)
	}
}

func corpusFile(t *testing.T, name string) []byte {
	t.Helper()
	dir := os.Getenv("DWG_TESTFILES_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "experiments", "dwg-reverse-engineering")
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Skipf("corpus unavailable: %v", err)
	}
	return data
}

// TestImportDWGPlanarRealFile imports a real planar drawing and checks it lands in
// a single 2D sketch on the chosen plane.
func TestImportDWGPlanarRealFile(t *testing.T) {
	part := compdef.NewPartComponentDefinition()
	res, err := ImportDWG(part, corpusFile(t, "testfile-7.dwg"), xyPlane(t))
	if err != nil {
		t.Fatalf("ImportDWG: %v", err)
	}
	if res.Is3D {
		t.Errorf("planar drawing imported as 3D")
	}
	if res.EntityCount < 1000 {
		t.Errorf("imported only %d entities", res.EntityCount)
	}
	if part.Sketches().Count() != 1 || part.Sketches3D().Count() != 0 {
		t.Errorf("expected one 2D sketch and no 3D sketch, got %d/%d", part.Sketches().Count(), part.Sketches3D().Count())
	}
}

// TestImportDWG3DRealFile imports a drawing with off-plane geometry and checks it
// lands in a 3D sketch.
func TestImportDWG3DRealFile(t *testing.T) {
	part := compdef.NewPartComponentDefinition()
	res, err := ImportDWG(part, corpusFile(t, "testfile-2.dwg"), xyPlane(t))
	if err != nil {
		t.Fatalf("ImportDWG: %v", err)
	}
	if !res.Is3D {
		t.Errorf("non-planar drawing imported as 2D")
	}
	if part.Sketches3D().Count() != 1 || part.Sketches().Count() != 0 {
		t.Errorf("expected one 3D sketch and no 2D sketch, got %d/%d", part.Sketches3D().Count(), part.Sketches().Count())
	}
}
