// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// TestCollapseDeadLoopDropsZeroLengthSegment pins the F6/T6/U3 fix: a loop carrying a coincident-point
// stub (ring[k]==ring[k+1] with a nil straight curve) drops exactly that segment from the ring and all
// four loop arrays, keeping them aligned — so no zero-length edge is minted to open the shell.
func TestCollapseDeadLoopDropsZeroLengthSegment(t *testing.T) {
	l := &filletLoop{
		pts:    []m.Point3{m.P3(0, 0, 0), m.P3(1, 0, 0), m.P3(1, 0, 0), m.P3(1, 1, 0)},
		curves: []geom.Curve3{nil, nil, nil, nil}, // segment 1 (B→B) is a nil straight zero-length stub
		srcV:   []uint64{10, 11, 12, 13},
		srcE:   []uint64{20, 21, 22, 23},
	}
	ring, survived := collapseDeadLoop(l, []int{0, 1, 1, 2}, 1e-6)
	if survived != -1 {
		t.Fatalf("a loop that collapses to 3 vertices must not be refused; survived = %d", survived)
	}
	if got := ring; len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("ring = %v, want [0 1 2]", got)
	}
	if len(l.srcV) != 3 || l.srcV[0] != 10 || l.srcV[1] != 12 || l.srcV[2] != 13 {
		t.Errorf("srcV = %v, want [10 12 13] (index 1 dropped)", l.srcV)
	}
	if len(l.srcE) != 3 || l.srcE[0] != 20 || l.srcE[1] != 22 || l.srcE[2] != 23 {
		t.Errorf("srcE = %v, want [20 22 23]", l.srcE)
	}
}

// TestCollapseDeadLoopKeepsFullCircleSeam guards the B1 boundary: a full-circle rim IS ring[k]==ring[k+1]
// (both ends weld to one vertex) but is a real edge — it must NOT be collapsed.
func TestCollapseDeadLoopKeepsFullCircleSeam(t *testing.T) {
	circle, err := geom.NewArc3d(m.P3(0, 0, 0), m.V3(0, 0, 1), m.V3(1, 0, 0), 5, 0, 2*math.Pi)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	l := &filletLoop{
		pts:    []m.Point3{m.P3(5, 0, 0), m.P3(5, 0, 0), m.P3(5, 0, 3), m.P3(5, 0, 3)},
		curves: []geom.Curve3{circle, nil, circle, nil},
		srcV:   []uint64{1, 2, 3, 4},
		srcE:   []uint64{5, 6, 7, 8},
	}
	// segments 0 and 2 are full-circle rims (ring self-loop, real curve); 1 and 3 join them.
	ring, survived := collapseDeadLoop(l, []int{0, 0, 1, 1}, 1e-6)
	if survived != -1 {
		t.Fatalf("full-circle seams must not be refused; survived = %d", survived)
	}
	if len(ring) != 4 {
		t.Fatalf("full-circle seams collapsed: ring len = %d, want 4 (kept)", len(ring))
	}
}

// TestCollapseDeadLoopCleanNoOp pins the no-op on a healthy loop (the 40 passing cases stay untouched).
func TestCollapseDeadLoopCleanNoOp(t *testing.T) {
	l := &filletLoop{
		pts:    []m.Point3{m.P3(0, 0, 0), m.P3(1, 0, 0), m.P3(1, 1, 0)},
		curves: []geom.Curve3{nil, nil, nil},
		srcV:   []uint64{1, 2, 3},
		srcE:   []uint64{4, 5, 6},
	}
	ring, survived := collapseDeadLoop(l, []int{0, 1, 2}, 1e-6)
	if survived != -1 {
		t.Fatalf("a clean loop must not be refused; survived = %d", survived)
	}
	if len(ring) != 3 || len(l.pts) != 3 {
		t.Errorf("clean loop mutated: ring=%v pts=%d", ring, len(l.pts))
	}
}

// TestCollapseDeadLoopRefusesDegenerateLoop is the #3389 regression: a loop that is nothing but dead
// zero-length segments collapses to fewer than three vertices, so it cannot bound a face. collapseDeadLoop
// must REFUSE it (survived < 3, the surviving count) and keep the original ring, so assembleBody can
// record a named defect here instead of deferring the malformed body to a later Validate.
func TestCollapseDeadLoopRefusesDegenerateLoop(t *testing.T) {
	// Three coincident points joined by nil zero-length stubs: every segment is dead.
	l := &filletLoop{
		pts:    []m.Point3{m.P3(2, 0, 0), m.P3(2, 0, 0), m.P3(2, 0, 0)},
		curves: []geom.Curve3{nil, nil, nil},
		srcV:   []uint64{1, 2, 3},
		srcE:   []uint64{4, 5, 6},
	}
	ring, survived := collapseDeadLoop(l, []int{0, 0, 0}, 1e-6)
	if survived < 0 || survived >= 3 {
		t.Fatalf("an all-dead loop must be refused with a surviving count < 3; survived = %d", survived)
	}
	if len(ring) != 3 {
		t.Errorf("a refused loop must keep its original ring for the builder; ring = %v", ring)
	}
}

// TestRecordDeadLoopRefusalsDefects proves the refusal reaches the body as a Defect (#3389): recording
// one refusal stamps the builder with a CodeAssembleDeadLoopCollapse Defect the built body carries.
func TestRecordDeadLoopRefusalsDefects(t *testing.T) {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("t", "body", 0)))
	recordDeadLoopRefusals(bld, []deadLoopRefusal{{face: 1, loop: 0, survived: 2, welded: 4}})
	body := bld.Build()
	found := false
	for _, d := range body.BuildDiagnostics() {
		if d.Code == CodeAssembleDeadLoopCollapse && d.Severity == diag.Defect {
			found = true
		}
	}
	if !found {
		t.Fatalf("a refused dead loop did not reach the body as a %q Defect: %v", CodeAssembleDeadLoopCollapse, body.BuildDiagnostics())
	}
}

// TestDeadCurve pins the discriminator: nil and zero-length curves are dead; a full-circle arc is not.
func TestDeadCurve(t *testing.T) {
	if !deadCurve(nil, 1e-6) {
		t.Error("nil curve must be dead")
	}
	circle, _ := geom.NewArc3d(m.P3(0, 0, 0), m.V3(0, 0, 1), m.V3(1, 0, 0), 5, 0, 2*math.Pi)
	if deadCurve(circle, 1e-6) {
		t.Error("a full-circle arc (midpoint 2·radius away) must NOT be dead")
	}
	stub, _ := geom.NewArc3d(m.P3(0, 0, 0), m.V3(0, 0, 1), m.V3(1, 0, 0), 5, 0, 1e-9)
	if !deadCurve(stub, 1e-6) {
		t.Error("a near-zero-sweep arc must be dead")
	}
}
