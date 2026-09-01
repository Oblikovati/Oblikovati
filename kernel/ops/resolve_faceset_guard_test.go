// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestResolveFaceSetRejectsAmbiguousKey is the ADR-0043 guard for face-set resolution (shell /
// draft): a key that binds to more than one face is a topological-naming collision and must fail
// honestly rather than silently shelling an unintended face. A clean key still resolves; a lost
// one reports honestly.
func TestResolveFaceSetRejectsAmbiguousKey(t *testing.T) {
	t.Parallel()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	mk := func(x, y float64, i int) *topo.Vertex {
		return bld.AddVertex(math.P3(x, y, 0), topo.NewLineage(topo.Tok("f", "vertex", i)))
	}
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	tri := func(a, b, c *topo.Vertex, e int, lin topo.Lineage) *topo.Face {
		ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, topo.NewLineage(topo.Tok("f", "edge", e)))
		bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, topo.NewLineage(topo.Tok("f", "edge", e+1)))
		ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, topo.NewLineage(topo.Tok("f", "edge", e+2)))
		return bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	}
	dup := topo.NewLineage(topo.Tok("f", "face", 0))
	f1 := tri(mk(0, 0, 0), mk(1, 0, 1), mk(0, 1, 2), 0, dup)
	tri(mk(3, 0, 3), mk(4, 0, 4), mk(3, 1, 5), 3, dup)
	uniq := tri(mk(6, 0, 6), mk(7, 0, 7), mk(6, 1, 8), 6, topo.NewLineage(topo.Tok("f", "face", 1)))
	body := bld.Build()

	if _, err := retopo.ResolveFaceSet(body, [][]byte{uniq.ReferenceKey()}); err != nil {
		t.Fatalf("unique face key should resolve, got %v", err)
	}
	_, err := retopo.ResolveFaceSet(body, [][]byte{f1.ReferenceKey()})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("ambiguous face key should be rejected, got %v", err)
	}
	if _, err := retopo.ResolveFaceSet(body, [][]byte{{0x01, 'z'}}); err == nil || !strings.Contains(err.Error(), "lost") {
		t.Errorf("missing face key should report lost, got %v", err)
	}
}
