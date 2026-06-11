// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The declared-kind audit of issue #145: every Sketch3DEntityKind declared in
// api/types must be reachable end-to-end — created over the wire, surviving a
// solve, and enumerated back by sketch3d.entities — or be explicitly tracked
// below as derived-by-another-method or as a sentinel. A kind added to
// api/types without an entry here fails the completeness check, the same guard
// the 3D constraint (#142) and dimension (#144) kinds already have.

// entity3DFixture creates one entity of a kind and names the kind that
// sketch3d.entities reports back for it. They differ only for bend, which
// materializes as an arc held by its maintaining bend constraint (#143).
type entity3DFixture struct {
	build          func(t *testing.T, r *Router, s *app.Session)
	enumeratedKind string
}

// addEntity3DJSON drives one sketch3d.addEntity call from raw arg fields and
// asserts the result echoes the requested kind.
func addEntity3DJSON(kind string, rest string) func(t *testing.T, r *Router, s *app.Session) {
	return func(t *testing.T, r *Router, s *app.Session) {
		t.Helper()
		var res wire.AddSketch3DEntityResult
		call(t, r, s, "sketch3d.addEntity",
			fmt.Sprintf(`{"sketchIndex":0,"kind":"%s",%s}`, kind, rest), &res)
		if res.Kind != kind {
			t.Errorf("addEntity result kind = %q, want %q", res.Kind, kind)
		}
	}
}

// selfEnumerating3D is the common case: the entity enumerates under the same
// kind it was created with.
func selfEnumerating3D(kind types.Sketch3DEntityKind, rest string) entity3DFixture {
	return entity3DFixture{build: addEntity3DJSON(string(kind), rest), enumeratedKind: string(kind)}
}

// bend3DFixture chains two lines into an L-corner and bends it; the bend
// entity is the fillet arc (enumerated "arc"), kept tangent by its constraint.
func bend3DFixture(t *testing.T, r *Router, s *app.Session) {
	t.Helper()
	var l1, l2 wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[1,0,0]]}`, &l1)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[1,0,0],[1,1,0]]}`, &l2)
	addEntity3DJSON("bend", fmt.Sprintf(`"lines":[%d,%d],"radius":"2.5 mm"`, l1.EntityID, l2.EntityID))(t, r, s)
}

// declared3DEntityFixtures maps every directly-creatable kind to its minimal
// sketch3d.addEntity fixture.
var declared3DEntityFixtures = map[types.Sketch3DEntityKind]entity3DFixture{
	types.Sketch3DEntityPoint:  selfEnumerating3D(types.Sketch3DEntityPoint, `"points":[[1,2,3]]`),
	types.Sketch3DEntityLine:   selfEnumerating3D(types.Sketch3DEntityLine, `"points":[[0,0,0],[1,0,0]]`),
	types.Sketch3DEntityCircle: selfEnumerating3D(types.Sketch3DEntityCircle, `"points":[[0,0,0]],"radius":"10 mm"`),
	types.Sketch3DEntityArc:    selfEnumerating3D(types.Sketch3DEntityArc, `"points":[[1,1,0],[1,0,0],[2,1,0]],"ccw":false`),
	types.Sketch3DEntityBend:   {build: bend3DFixture, enumeratedKind: string(types.Sketch3DEntityArc)},
	types.Sketch3DEntityEllipse: selfEnumerating3D(types.Sketch3DEntityEllipse,
		`"points":[[0,0,0]],"majorRadius":"50 mm","minorRadius":"30 mm"`),
	types.Sketch3DEntityEllipticalArc: selfEnumerating3D(types.Sketch3DEntityEllipticalArc,
		`"points":[[0,0,0]],"majorRadius":"40 mm","minorRadius":"20 mm","startAngle":"0 deg","sweepAngle":"90 deg"`),
	types.Sketch3DEntitySpline: selfEnumerating3D(types.Sketch3DEntitySpline,
		`"points":[[0,0,0],[1,2,0],[3,1,1]]`),
	types.Sketch3DEntityControlPointSpline: selfEnumerating3D(types.Sketch3DEntityControlPointSpline,
		`"points":[[0,0,0],[1,0,2],[2,0,0]]`),
	types.Sketch3DEntityFixedSpline: selfEnumerating3D(types.Sketch3DEntityFixedSpline,
		`"points":[[0,0,0],[1,1,1],[2,0,1]]`),
	types.Sketch3DEntityEquationCurve: selfEnumerating3D(types.Sketch3DEntityEquationCurve,
		`"xExpr":"cos(t)","yExpr":"sin(t)","zExpr":"t","t0":0,"t1":3.14159`),
	types.Sketch3DEntityHelical: selfEnumerating3D(types.Sketch3DEntityHelical,
		`"points":[[0,0,1]],"radius":"4 mm","mode":"pitchRevolution","pitch":"10 mm","revolutions":3`),
}

// derived3DEntityKinds are declared kinds NOT built by sketch3d.addEntity: each
// is produced by its own wire method against host geometry and carries its own
// end-to-end test (TestSketch3DSurfaceCurves, TestSketch3DInclude).
var derived3DEntityKinds = map[types.Sketch3DEntityKind]string{
	types.Sketch3DEntityIntersection:     wire.MethodSketch3DAddSurfaceCurve,
	types.Sketch3DEntityOnFace:           wire.MethodSketch3DAddSurfaceCurve,
	types.Sketch3DEntityProjectToSurface: wire.MethodSketch3DAddSurfaceCurve,
	types.Sketch3DEntitySilhouette:       wire.MethodSketch3DAddSurfaceCurve,
	types.Sketch3DEntityOffset:           wire.MethodSketch3DAddSurfaceCurve,
	types.Sketch3DEntityIncludedPoint:    wire.MethodSketch3DInclude,
	types.Sketch3DEntityIncludedCurve:    wire.MethodSketch3DInclude,
}

// sentinel3DEntityKinds never name constructible geometry.
var sentinel3DEntityKinds = map[types.Sketch3DEntityKind]string{
	types.Sketch3DEntityUnknown: "fallback for entities the enumeration cannot classify",
}

// allDeclared3DEntityKinds mirrors the const block in api/types/sketch3d.go
// (Go cannot enumerate constants); keep in sync when the API grows.
var allDeclared3DEntityKinds = []types.Sketch3DEntityKind{
	types.Sketch3DEntityPoint, types.Sketch3DEntityLine, types.Sketch3DEntityCircle,
	types.Sketch3DEntityArc, types.Sketch3DEntityBend, types.Sketch3DEntityEllipse,
	types.Sketch3DEntityEllipticalArc, types.Sketch3DEntitySpline,
	types.Sketch3DEntityControlPointSpline, types.Sketch3DEntityFixedSpline,
	types.Sketch3DEntityEquationCurve, types.Sketch3DEntityHelical,
	types.Sketch3DEntityIntersection, types.Sketch3DEntityOnFace,
	types.Sketch3DEntityProjectToSurface, types.Sketch3DEntitySilhouette,
	types.Sketch3DEntityOffset, types.Sketch3DEntityIncludedPoint,
	types.Sketch3DEntityIncludedCurve, types.Sketch3DEntityUnknown,
}

// TestEvery3DEntityKindAccepted drives sketch3d.addEntity for every fixture
// kind, then solves and re-enumerates: the entity must survive the solve and
// come back under its expected kind.
func TestEvery3DEntityKindAccepted(t *testing.T) {
	for kind, fixture := range declared3DEntityFixtures {
		t.Run(string(kind), func(t *testing.T) {
			r, s := emptyPartSession(t)
			call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
			fixture.build(t, r, s)
			var solved wire.SolveSketch3DResult
			call(t, r, s, "sketch3d.solve", `{"sketchIndex":0}`, &solved)
			if !solved.Converged || !solved.Healthy {
				t.Errorf("solve with a %s present = %+v, want converged and healthy", kind, solved)
			}
			assertEnumerated3DKind(t, r, s, fixture.enumeratedKind)
		})
	}
}

// assertEnumerated3DKind fails unless sketch3d.entities reports at least one
// entity of the given kind.
func assertEnumerated3DKind(t *testing.T, r *Router, s *app.Session, kind string) {
	t.Helper()
	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	for _, e := range ents.Entities {
		if e.Kind == kind {
			return
		}
	}
	t.Errorf("enumerated entities = %+v, want at least one of kind %q", ents.Entities, kind)
}

// TestDeclared3DEntityKindsComplete fails when a declared kind is not exactly
// one of: fixture-covered, derived-by-another-method, or sentinel — the guard
// that api/types can never advertise 3D entity kinds the host cannot build.
func TestDeclared3DEntityKindsComplete(t *testing.T) {
	for _, kind := range allDeclared3DEntityKinds {
		entries := 0
		if _, ok := declared3DEntityFixtures[kind]; ok {
			entries++
		}
		if _, ok := derived3DEntityKinds[kind]; ok {
			entries++
		}
		if _, ok := sentinel3DEntityKinds[kind]; ok {
			entries++
		}
		if entries != 1 {
			t.Errorf("kind %q has %d coverage entries, want exactly 1 (fixture, derived, or sentinel)", kind, entries)
		}
	}
}
