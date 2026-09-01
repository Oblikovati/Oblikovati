// SPDX-License-Identifier: GPL-2.0-only

package transform_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/test-utilities/opfixture"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// cavityBody and squareWireBody are restated from kernel/ops' test package: an internal
// test helper cannot be imported, and a shared fixture package would import kernel/ops,
// which kernel/ops' own tests could then not use (import cycle). This is the test
// scaffolding sonar.cpd.exclusions already accounts for.

// squareWireBody attaches a unit-square wire (CCW in the XY plane) to a body.
func squareWireBody(side float64) (*topo.Body, *topo.Wire) {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("w", "body", 0)))
	p := []math.Point3{
		math.P3(0, 0, 0), math.P3(math.Scalar(side), 0, 0),
		math.P3(math.Scalar(side), math.Scalar(side), 0), math.P3(0, math.Scalar(side), 0),
	}
	v := make([]*topo.Vertex, 4)
	for i := range p {
		v[i] = bld.AddVertex(p[i], topo.NewLineage(topo.Tok("w", "vertex", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range 4 {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(p[i], p[j]), v[i], v[j], topo.NewLineage(topo.Tok("w", "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	body := bld.Build()
	w := body.AttachWire(topo.NewLineage(topo.Tok("w", "wire", 0)), uses)
	return body, w
}

// TransformBody historically rejected multi-shell bodies and silently dropped
// each face's material sense; both broke transformed cavity bodies (#629/#628
// Copy rides on this). The cavity body is the regression vehicle: 4³ − 2³ = 56
// of material in two shells with reversed cavity walls.

func TestTransformBodyCavityPreservesShellsAndVolume(t *testing.T) {
	t.Parallel()
	body := opfixture.Cavity(t)
	moved, err := transform.TransformBody(body, math.Translation4(math.V3(10, 0, 0)), func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	if got := len(moved.Shells()); got != 2 {
		t.Errorf("moved cavity body has %d shells, want 2", got)
	}
	if v := query.BodyGeometryProperties(moved, ops.DefaultQuality()).Volume; stdmath.Abs(v-56) > 0.1 {
		t.Errorf("moved cavity volume = %g, want 56 (face sense must survive)", v)
	}
	if r := ops.Validate(moved); !r.Valid {
		t.Errorf("moved cavity invalid: %v", r.Issues)
	}
}

// TestTransformBodyCarriesWires: a wire-only body transforms with its wire.
func TestTransformBodyCarriesWires(t *testing.T) {
	t.Parallel()
	_, w := squareWireBody(1)
	src := w.Body()
	moved, err := transform.TransformBody(src, math.Translation4(math.V3(0, 0, 5)), func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	if len(moved.Wires()) != 1 {
		t.Fatalf("moved body has %d wires, want 1", len(moved.Wires()))
	}
	p := moved.Wires()[0].Edges()[0].StartVertex().Point()
	if stdmath.Abs(float64(p.Z)-5) > 1e-12 {
		t.Errorf("moved wire start Z = %v, want 5", p.Z)
	}
}
