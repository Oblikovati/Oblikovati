// SPDX-License-Identifier: GPL-2.0-only

package blend_test

import (
	"testing"

	"oblikovati.org/kernel/blend"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	opsblend "oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// blendRoundedBox fillets the four vertical edges of a box, yielding the tangent-loop top rim used
// to exercise the not-known-part (multi-edge) classification.
func blendRoundedBox(box *topo.Body) (*topo.Body, error) {
	var keys [][]byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return opsblend.FilletEdges(box, keys, 0.5)
}

// fakeSolver is a named stand-in for the Phase-4 marcher, so the builder's dispatch can be tested
// without any real geometry.
type fakeSolver struct{ res blend.Result }

func (f fakeSolver) March(*blend.Spine, blend.SectionFunctional) blend.Result { return f.res }

// TestClassifyKnownPartMatchesRouting checks the known-part classifier agrees with the analytic
// routing ops already performs: a straight box edge is a planar-edge cylinder blend, a cylinder rim
// is a toroidal band, and a multi-edge tangent loop is not a known part (it needs the marcher).
func TestClassifyKnownPartMatchesRouting(t *testing.T) {
	box := spineBox(2, 2, 2)
	boxSpine, _ := blend.NewSpine([]*topo.Edge{box.Edges()[0]}, false)
	if k := blend.ClassifyKnownPart(boxSpine); k != blend.KnownPlanarEdge {
		t.Errorf("box edge classified %v, want planar-edge", k)
	}

	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	rimSpine, _ := blend.NewSpine([]*topo.Edge{circleEdge(t, cyl)}, true)
	if k := blend.ClassifyKnownPart(rimSpine); k != blend.KnownCylinderRim {
		t.Errorf("cylinder rim classified %v, want cylinder-rim", k)
	}

	rounded, err := blendRoundedBox(box)
	if err != nil {
		t.Fatal(err)
	}
	loop, closed := rimChainEdges(t, rounded, 2.0)
	loopSpine, _ := blend.NewSpine(loop, closed)
	if k := blend.ClassifyKnownPart(loopSpine); k != blend.NotKnownPart {
		t.Errorf("8-edge tangent loop classified %v, want not-known-part", k)
	}
}

// TestSectionFunctionalCarriers checks the section family flags and extents distinguish fillet from
// chamfer — the single parameter the builder routes on.
func TestSectionFunctionalCarriers(t *testing.T) {
	fil := blend.ConstRadiusFillet{R: 2}
	if fil.IsChamfer() || fil.Extent(0) != 2 {
		t.Errorf("fillet section: chamfer=%v extent=%g, want false/2", fil.IsChamfer(), fil.Extent(0))
	}
	ch := blend.SymmetricChamfer{D: 1.5}
	if !ch.IsChamfer() || ch.Extent(0) != 1.5 {
		t.Errorf("chamfer section: chamfer=%v extent=%g, want true/1.5", ch.IsChamfer(), ch.Extent(0))
	}
}

// TestBuilderDispatch checks the skeleton: Plan classifies, March with no solver reports
// NotImplemented (nothing built), and March with an injected solver returns its result.
func TestBuilderDispatch(t *testing.T) {
	box := spineBox(2, 2, 2)
	sp, _ := blend.NewSpine([]*topo.Edge{box.Edges()[0]}, false)

	stub := blend.NewBuilder(nil)
	if k := stub.Plan(sp); k != blend.KnownPlanarEdge {
		t.Errorf("Plan = %v, want planar-edge", k)
	}
	res := stub.March(sp, blend.ConstRadiusFillet{R: 1})
	if res.Status != blend.StatusNotImplemented || res.HasResult() || !res.BadShape() {
		t.Errorf("nil-solver March = %+v, want NotImplemented/no-result/bad-shape", res)
	}

	want := blend.Result{Segments: []blend.BlendSegment{{}}, Status: blend.StatusOk}
	wired := blend.NewBuilder(fakeSolver{res: want})
	if got := wired.March(sp, blend.ConstRadiusFillet{R: 1}); !got.HasResult() || got.BadShape() {
		t.Errorf("wired March = %+v, want a usable result", got)
	}
}

// circleEdge returns the first edge of b whose geometry is a full circle (a cylinder rim).
func circleEdge(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.Circle); ok {
			return e
		}
	}
	t.Fatal("no circular edge")
	return nil
}

// TestMarchCylinderRimTorus drives the full SegmentSolver.March path on a real spine: a cylinder's
// rim (a plane+cylinder support pair) marches to a single torus blend segment — the curved-neighbour
// case the analytic catalog could not route from a general fillet (#1806).
func TestMarchCylinderRimTorus(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	sp, _ := blend.NewSpine([]*topo.Edge{circleEdge(t, cyl)}, true)
	m := &blend.Marcher{
		Inside: func(p math.Point3) bool { return ops.PointInsideBody(cyl, p) },
		Res:    geom.ResolutionForBox(cyl.RangeBox()),
	}
	res := m.March(sp, blend.ConstRadiusFillet{R: 0.5})
	if res.Status != blend.StatusOk || !res.HasResult() || len(res.Segments) != 1 {
		t.Fatalf("march = status %v, %d segments; want one Ok segment", res.Status, len(res.Segments))
	}
	if _, ok := res.Segments[0].Surface.(geom.Torus); !ok {
		t.Errorf("rim blend surface is %T, want a torus", res.Segments[0].Surface)
	}
}
