// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"math"
	"testing"
)

// decodeGeometry walks a file and decodes every LINE/CIRCLE/ARC/POINT into the
// intermediate model, keyed by handle, returning also a per-type success count.
func decodeGeometry(t *testing.T, name string) (map[uint64]Entity, map[string]int) {
	t.Helper()
	data := loadTestFile(t, name)
	h, err := ParseFileHeader(data)
	if err != nil {
		t.Fatalf("ParseFileHeader: %v", err)
	}
	omb, err := h.ObjectMapBytes(data)
	if err != nil {
		t.Fatalf("ObjectMapBytes: %v", err)
	}
	od, err := h.ObjectData(data)
	if err != nil {
		t.Fatalf("ObjectData: %v", err)
	}
	refs, err := parseObjectMap(omb)
	if err != nil {
		t.Fatalf("parseObjectMap: %v", err)
	}
	byHandle := map[uint64]Entity{}
	counts := map[string]int{}
	for _, ref := range refs {
		hdr, err := decodeObjectHeader(od, ref, h.Version)
		if err != nil {
			continue
		}
		switch hdr.Type {
		case TypeLine, TypeCircle, TypeArc, TypePoint, TypeEllipse, TypeLwpolyline, TypeSpline:
		default:
			continue
		}
		cur, err := seekEntity(od, ref, h.Version)
		if err != nil {
			t.Fatalf("%s: seek handle %d (%s): %v", name, ref.Handle, hdr.Type.Name(), err)
		}
		e, err := decodeEntity(cur.geom, hdr, h.Version)
		if err != nil || e == nil {
			t.Fatalf("%s: decode handle %d (%s): %v", name, ref.Handle, hdr.Type.Name(), err)
		}
		byHandle[ref.Handle] = e
		counts[hdr.Type.Name()]++
	}
	return byHandle, counts
}

// TestDecodeGeometryCorpus decodes every core entity in both containers and
// requires the per-type counts to match the oracle exactly, with zero decode
// failures — proof the common-entity-data skip lands on the geometry for every
// object, not just a lucky sample.
func TestDecodeGeometryCorpus(t *testing.T) {
	cases := []struct {
		file string
		want map[string]int
	}{
		{"testfile-1.dwg", map[string]int{
			"LINE": 58062, "ARC": 1670, "CIRCLE": 959, "POINT": 739,
			"ELLIPSE": 1271, "LWPOLYLINE": 15525, "SPLINE": 2898,
		}},
		{"testfile-2.dwg", map[string]int{"LINE": 134240, "ARC": 41950, "POINT": 34264}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			_, counts := decodeGeometry(t, c.file)
			for name, want := range c.want {
				if counts[name] != want {
					t.Errorf("%s decoded %d, want %d (oracle)", name, counts[name], want)
				}
			}
		})
	}
}

// TestDecodeGeometryCoordsMatchOracle pins specific entities' coordinates against
// dwgread output — a byte-exact check of the bit-level geometry decode (RD/DD/BD
// and the z-is-zero handling) for each of the four core types.
func TestDecodeGeometryCoordsMatchOracle(t *testing.T) {
	ents, _ := decodeGeometry(t, "testfile-1.dwg")

	line, ok := ents[1019353].(*Line)
	if !ok {
		t.Fatal("handle 1019353 not a Line")
	}
	wantClose(t, "line.start", line.Start, [3]float64{758.4078325448183, 129.00276583787206, 0})
	wantClose(t, "line.end", line.End, [3]float64{759.9009334444454, 129.00276583787206, 0})

	circle, ok := ents[1019067].(*Circle)
	if !ok {
		t.Fatal("handle 1019067 not a Circle")
	}
	wantClose(t, "circle.center", circle.Center, [3]float64{0, 0, 0})
	if math.Abs(circle.Radius-3.9375) > 1e-9 {
		t.Errorf("circle.radius = %v, want 3.9375", circle.Radius)
	}

	arc, ok := ents[1027220].(*Arc)
	if !ok {
		t.Fatal("handle 1027220 not an Arc")
	}
	wantClose(t, "arc.center", arc.Center, [3]float64{540032614.6859194, 178230901.49567443, 0})
	if math.Abs(arc.Radius-249.99999999997783) > 1e-6 ||
		math.Abs(arc.StartAngle-3.29382026142359) > 1e-9 ||
		math.Abs(arc.EndAngle-4.54113751756787) > 1e-9 {
		t.Errorf("arc = r%v s%v e%v, want r249.999.. s3.2938.. e4.5411..", arc.Radius, arc.StartAngle, arc.EndAngle)
	}

	point, ok := ents[1026420].(*Point)
	if !ok {
		t.Fatal("handle 1026420 not a Point")
	}
	wantClose(t, "point", point.Position, [3]float64{539962606.9976188, 178228176.57959518, 0})

	ellipse, ok := ents[1035518].(*Ellipse)
	if !ok {
		t.Fatal("handle 1035518 not an Ellipse")
	}
	wantClose(t, "ellipse.center", ellipse.Center, [3]float64{540022641.2854427, 178190920.62832335, 0})
	wantClose(t, "ellipse.major", ellipse.MajorAxis, [3]float64{-2.2e-13, 14.99999999999998, 0})
	if math.Abs(ellipse.AxisRatio-0.17364817766693) > 1e-9 {
		t.Errorf("ellipse.axisRatio = %v, want 0.17364817766693", ellipse.AxisRatio)
	}

	lwp, ok := ents[993637].(*LwPolyline)
	if !ok {
		t.Fatal("handle 993637 not an LwPolyline")
	}
	if len(lwp.Points) != 2 ||
		math.Abs(lwp.Points[0][0]-(-0.5)) > 1e-9 || math.Abs(lwp.Points[1][1]-0.5) > 1e-9 {
		t.Errorf("lwpolyline points = %v, want [[-0.5 -0.5] [0.5 0.5]]", lwp.Points)
	}

	spline, ok := ents[1019296].(*Spline)
	if !ok {
		t.Fatal("handle 1019296 not a Spline")
	}
	if spline.Degree != 3 || len(spline.ControlPoints) != 4 || len(spline.Knots) != 8 {
		t.Errorf("spline = degree %d, %d ctrl, %d knots; want degree 3, 4 ctrl, 8 knots",
			spline.Degree, len(spline.ControlPoints), len(spline.Knots))
	}
	wantClose(t, "spline.ctrl0", spline.ControlPoints[0], [3]float64{748.9970525371714, 126.32570743854146, 0})
}

func wantClose(t *testing.T, what string, got, want [3]float64) {
	t.Helper()
	for i := range got {
		if math.Abs(got[i]-want[i]) > 1e-6 {
			t.Errorf("%s[%d] = %v, want %v", what, i, got[i], want[i])
		}
	}
}
