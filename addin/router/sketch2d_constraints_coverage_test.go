// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The 2D twin of the sketch3d declared-kind coverage guard (#1643, audit S8): every
// GeometricConstraintKind declared in api/types must be accepted end-to-end by
// sketch.addConstraint, or carry an explicit justification below — never a silent gap.
//
// Retro-verification: run against the pre-#1574 code shape this fails on "symmetry"
// (enumerable-but-not-creatable — the exact bug class this guard exists to prevent),
// and on the pre-#1643 shape it fails on "smooth" (the gap closed alongside this file).

// constraint2DFixture builds the minimal geometry for one kind and returns the
// addConstraint entity refs (plus optional clientId/name for the custom kind).
type constraint2DFixture func(t *testing.T, r *Router, s *app.Session) []uint64

// addEntity2D posts one sketch.addEntity call and returns the new entity's id.
func addEntity2D(t *testing.T, r *Router, s *app.Session, args string) uint64 {
	t.Helper()
	var res wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", args, &res)
	return res.EntityID
}

// twoLines2D builds two non-parallel lines and returns their entity ids.
func twoLines2D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	return []uint64{
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[10,0],[11,0]]}`),
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[10,1],[11,2]]}`),
	}
}

// nPoints2D builds n standalone points and returns their ids.
func nPoints2D(n int) constraint2DFixture {
	return func(t *testing.T, r *Router, s *app.Session) []uint64 {
		t.Helper()
		out := make([]uint64, n)
		for i := range out {
			out[i] = addEntity2D(t, r, s,
				fmt.Sprintf(`{"sketchIndex":0,"kind":"point","points":[[%d,%d]]}`, 10+i, 2*i))
		}
		return out
	}
}

// pointAndLine2D builds a standalone point and a line, point first.
func pointAndLine2D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	return []uint64{
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[10.5,0.5]]}`),
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[10,0],[11,0]]}`),
	}
}

// pointAndCircle2D builds a standalone point and a circle, point first.
func pointAndCircle2D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	return []uint64{
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"point","points":[[12,0]]}`),
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[10,0]],"radius":"2 cm"}`),
	}
}

// twoCircles2D builds two circles of different radii and returns their ids.
func twoCircles2D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	return []uint64{
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[10,0]],"radius":"2 cm"}`),
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[16,0]],"radius":"1 cm"}`),
	}
}

// lineAndCircle2D builds a line clear of a circle and returns [line, circle].
func lineAndCircle2D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	return []uint64{
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[10,5],[16,5]]}`),
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[10,0]],"radius":"2 cm"}`),
	}
}

// symmetryTrio2D builds two points and the mirror line, in the (a, b, about) order
// symmetryConstraint expects.
func symmetryTrio2D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	pts := nPoints2D(2)(t, r, s)
	return append(pts, addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[13,-1],[13,1]]}`))
}

// lineAndSpline2D builds a line and a fit spline starting near the line's end —
// the smooth (G2) fixture; at least one side must be a spline.
func lineAndSpline2D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	return []uint64{
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"line","points":[[9,0],[10,0]]}`),
		addEntity2D(t, r, s, `{"sketchIndex":0,"kind":"spline","points":[[10.2,0.1],[11,1],[12,0]]}`),
	}
}

// declared2DConstraintFixtures maps every wire-creatable kind to its minimal fixture.
var declared2DConstraintFixtures = map[types.GeometricConstraintKind]constraint2DFixture{
	types.GeoConstraintCoincident:    nPoints2D(2),
	types.GeoConstraintPointOnLine:   pointAndLine2D,
	types.GeoConstraintMidpoint:      pointAndLine2D,
	types.GeoConstraintPointOnCircle: pointAndCircle2D,
	types.GeoConstraintHorizontal:    nPoints2D(2),
	types.GeoConstraintVertical:      nPoints2D(2),
	types.GeoConstraintParallel:      twoLines2D,
	types.GeoConstraintPerpendicular: twoLines2D,
	types.GeoConstraintCollinear:     twoLines2D,
	types.GeoConstraintConcentric:    twoCircles2D,
	types.GeoConstraintEqualLength:   twoLines2D,
	types.GeoConstraintEqualRadius:   twoCircles2D,
	types.GeoConstraintTangent:       lineAndCircle2D,
	types.GeoConstraintSymmetry:      symmetryTrio2D,
	types.GeoConstraintFix:           nPoints2D(1),
	types.GeoConstraintSmooth:        lineAndSpline2D,
	types.GeoConstraintGround:        nPoints2D(1),
	types.GeoConstraintOffset:        twoLines2D,
	types.GeoConstraintPattern:       nPoints2D(2),
	types.GeoConstraintCustom:        nPoints2D(2),
}

// intentionallyNotWireCreatable2D are declared kinds that sketch.addConstraint must NOT
// accept, each with the reason — the policy mirror of the 3D guard's tracked list.
var intentionallyNotWireCreatable2D = map[types.GeometricConstraintKind]string{
	// The text-box anchor is auto-created with its text box and refuses standalone
	// deletion (M06-F11, #626); standalone creation would orphan it.
	types.GeoConstraintTextBox: "auto-created with its text box (M06-F11)",
	// The sentinel for model constraints with no wire mapping; never a real kind.
	types.GeoConstraintUnknown: "enumeration sentinel, not a constraint",
}

// allDeclared2DConstraintKinds mirrors the const block in api/types/sketch.go
// (Go cannot enumerate constants); keep in sync when the API grows.
var allDeclared2DConstraintKinds = []types.GeometricConstraintKind{
	types.GeoConstraintCoincident, types.GeoConstraintPointOnLine, types.GeoConstraintMidpoint,
	types.GeoConstraintPointOnCircle, types.GeoConstraintHorizontal, types.GeoConstraintVertical,
	types.GeoConstraintParallel, types.GeoConstraintPerpendicular, types.GeoConstraintCollinear,
	types.GeoConstraintConcentric, types.GeoConstraintEqualLength, types.GeoConstraintEqualRadius,
	types.GeoConstraintTangent, types.GeoConstraintSymmetry, types.GeoConstraintFix,
	types.GeoConstraintSmooth, types.GeoConstraintGround, types.GeoConstraintOffset,
	types.GeoConstraintPattern, types.GeoConstraintTextBox, types.GeoConstraintCustom,
	types.GeoConstraintUnknown,
}

// TestEvery2DConstraintKindAccepted drives sketch.addConstraint for every declared kind
// with a fixture and asserts the constraint lands with the same kind enumerated back by
// sketch.constraints.
func TestEvery2DConstraintKindAccepted(t *testing.T) {
	for kind, fixture := range declared2DConstraintFixtures {
		t.Run(string(kind), func(t *testing.T) {
			r, s := seededSession(t)
			refs := fixture(t, r, s)
			var res wire.AddConstraintResult
			call(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
				SketchIndex: 0, Kind: string(kind), Entities: refs,
				ClientID: "coverage-test", Name: "tag", // read only by the custom kind
			}), &res)
			if res.Kind != string(kind) {
				t.Errorf("addConstraint kind = %q, want %q", res.Kind, kind)
			}
			assertEnumerated2DKind(t, r, s, kind)
		})
	}
}

// assertEnumerated2DKind fails unless sketch.constraints lists at least one constraint
// of the given kind (the custom kind enumerates per its add-in mapping, checked too).
func assertEnumerated2DKind(t *testing.T, r *Router, s *app.Session, kind types.GeometricConstraintKind) {
	t.Helper()
	var cons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	for _, c := range cons.Constraints {
		if c.Kind == string(kind) {
			return
		}
	}
	t.Errorf("sketch.constraints lists no %q constraint after addConstraint accepted it", kind)
}

// TestNotWireCreatable2DKindsRefused pins the policy list: the kinds justified above
// must actually be refused, so a future implementation removes the entry deliberately.
func TestNotWireCreatable2DKindsRefused(t *testing.T) {
	for kind, why := range intentionallyNotWireCreatable2D {
		t.Run(string(kind), func(t *testing.T) {
			r, s := seededSession(t)
			refs := nPoints2D(2)(t, r, s)
			err := tryCall(t, r, s, "sketch.addConstraint", mustJSON(t, wire.AddConstraintArgs{
				SketchIndex: 0, Kind: string(kind), Entities: refs,
			}))
			if err == nil {
				t.Errorf("kind %q (%s) was accepted over the wire — implement it deliberately and "+
					"move it to declared2DConstraintFixtures", kind, why)
			}
		})
	}
}

// TestDeclared2DConstraintKindsComplete fails when a declared kind is neither covered by
// a fixture nor justified as not-wire-creatable — the guard that api/types can never
// again advertise 2D kinds the vertical does not deliver (#1574's bug class).
func TestDeclared2DConstraintKindsComplete(t *testing.T) {
	for _, kind := range allDeclared2DConstraintKinds {
		_, covered := declared2DConstraintFixtures[kind]
		why, excluded := intentionallyNotWireCreatable2D[kind]
		switch {
		case covered && excluded:
			t.Errorf("kind %q is both covered and excluded (%s) — remove one", kind, why)
		case !covered && !excluded:
			t.Errorf("kind %q is declared in api/types but has no fixture and no justification", kind)
		}
	}
}
