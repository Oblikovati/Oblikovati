// SPDX-License-Identifier: GPL-2.0-only

package proxy

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
)

// Real definition-space entities satisfy the proxy capability constraints, so the
// generic mechanism proxies them with no per-entity code. These compile-time
// assertions are the guarantee that a part's face/edge/vertex/body and the work
// features can be proxied — proven without constructing any geometry.
var (
	_ Boxed    = (*topo.Face)(nil)
	_ Boxed    = (*topo.Edge)(nil)
	_ Boxed    = (*topo.Vertex)(nil)
	_ Boxed    = (*topo.Body)(nil)
	_ Pointed  = (*topo.Vertex)(nil)
	_ Pointed  = (*feature.WorkPoint)(nil)
	_ Directed = (*feature.WorkAxis)(nil)
)

// fakeDef is a minimal occurrence.Definition so tests can build occurrences with a
// known transform; its box is irrelevant to the proxy (the proxy transforms the
// proxied entity, not the occurrence's own definition).
type fakeDef struct{}

func (fakeDef) RangeBox() math.Box { return math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1)) }

// boxedEntity stands in for a topo face/edge/vertex (proven Boxed by the assertions
// above), reporting a fixed definition-space range box.
type boxedEntity struct{ box math.Box }

func (b boxedEntity) RangeBox() math.Box { return b.box }

// pointedEntity stands in for a vertex / work point; directedEntity for a work axis.
type pointedEntity struct{ p math.Point3 }

func (e pointedEntity) Point() math.Point3 { return e.p }

type directedEntity struct{ d math.UnitVector3 }

func (e directedEntity) Direction() math.UnitVector3 { return e.d }

// occAt returns a single occurrence placed at translation t, for building contexts.
func occAt(t math.Vector3) *occurrence.Occurrence {
	return occurrence.NewOccurrences().AddByComponentDefinition("c:1", fakeDef{}, math.Translation4(t))
}

// TestProxyReportsAssemblySpaceBox is the F03 acceptance: an entity (here a face)
// viewed via an occurrence reports its geometry in assembly space, not part space,
// while Native recovers the unchanged underlying entity. (Proxy[E] is a distinct type
// from E, so part-space geometry cannot be passed where an assembly-space proxy is
// required — a compile-time guarantee, not exercised at runtime.)
func TestProxyReportsAssemblySpaceBox(t *testing.T) {
	face := boxedEntity{box: math.NewBox(math.P3(0, 0, 0), math.P3(2, 1, 1))}
	ctx := NewContext(occAt(math.V3(10, 0, 0)))
	p := CreateGeometryProxy(ctx, face)

	got := RangeBoxInContext(p)
	if got.Min != (math.P3(10, 0, 0)) || got.Max != (math.P3(12, 1, 1)) {
		t.Errorf("assembly-space box = %v..%v, want {10 0 0}..{12 1 1}", got.Min, got.Max)
	}
	if p.Native().RangeBox().Max != (math.P3(2, 1, 1)) {
		t.Errorf("native box = %v, want the unchanged part-space {2 1 1}", p.Native().RangeBox().Max)
	}
}

func TestPointAndDirectionInContext(t *testing.T) {
	ctx := NewContext(occAt(math.V3(5, 0, 0)))
	pt := CreateGeometryProxy(ctx, pointedEntity{p: math.P3(1, 2, 3)})
	if got := PointInContext(pt); got != (math.P3(6, 2, 3)) {
		t.Errorf("point in context = %v, want {6 2 3}", got)
	}
	// A pure translation leaves directions unchanged.
	xdir, _ := math.NewUnitVector3(1, 0, 0)
	dir := CreateGeometryProxy(ctx, directedEntity{d: xdir})
	got, ok := DirectionInContext(dir)
	if !ok || got.Dot(xdir) < 0.999999 {
		t.Errorf("direction in context = %v ok=%v, want unchanged +X", got, ok)
	}
}

// TestPathContextComposesNestedTransforms proves nested occurrence paths compose: a
// point in a pin's space, where the pin sits +1x inside a sub-assembly that sits +10x
// in the top assembly, lands at +11x in the top.
func TestPathContextComposesNestedTransforms(t *testing.T) {
	g1 := occAt(math.V3(10, 0, 0))
	pin := occAt(math.V3(1, 0, 0))
	ctx := NewPathContext([]*occurrence.Occurrence{g1, pin})
	if ctx.Occurrence() != pin {
		t.Error("path context should identify the leaf occurrence")
	}
	p := CreateGeometryProxy(ctx, pointedEntity{p: math.P3(0, 0, 0)})
	if got := PointInContext(p); got != (math.P3(11, 0, 0)) {
		t.Errorf("nested point = %v, want {11 0 0} (10+1 composed)", got)
	}
}

func TestEmptyPathContextIsIdentity(t *testing.T) {
	ctx := NewPathContext(nil)
	if ctx.Occurrence() != nil {
		t.Error("empty path context should have no occurrence")
	}
	p := CreateGeometryProxy(ctx, pointedEntity{p: math.P3(7, 8, 9)})
	if got := PointInContext(p); got != (math.P3(7, 8, 9)) {
		t.Errorf("identity context moved the point to %v, want {7 8 9}", got)
	}
}
