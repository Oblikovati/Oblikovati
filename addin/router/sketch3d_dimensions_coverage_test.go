// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The declared-kind coverage guard for 3D dimensions (issue #144), mirroring the
// constraint guard: every Dimension3DConstraintKind declared in api/types must be
// accepted end-to-end by sketch3d.addDimension or carry a tracking issue.

// dimension3DFixture builds the geometry for one kind and returns the addDimension
// request (entities + expression + optional plane).
type dimension3DFixture func(t *testing.T, r *Router, s *app.Session) string

// declared3DDimensionFixtures maps EVERY declared kind to its request fixture.
var declared3DDimensionFixtures = map[types.Dimension3DConstraintKind]dimension3DFixture{
	types.Dim3DDistance: func(t *testing.T, r *Router, s *app.Session) string {
		ids := nPoints3D(2)(t, r, s)
		return fmt.Sprintf(`{"sketchIndex":0,"kind":"distance","entities":[%d,%d],"expression":"30 mm"}`, ids[0], ids[1])
	},
	types.Dim3DLineLength: func(t *testing.T, r *Router, s *app.Session) string {
		ids := oneLine3D(t, r, s)
		return fmt.Sprintf(`{"sketchIndex":0,"kind":"lineLength","entities":[%d],"expression":"10 mm"}`, ids[0])
	},
	types.Dim3DRadius: func(t *testing.T, r *Router, s *app.Session) string {
		ids := twoCircles3D(t, r, s)
		return fmt.Sprintf(`{"sketchIndex":0,"kind":"radius","entities":[%d],"expression":"10 mm"}`, ids[0])
	},
	types.Dim3DPointPlaneDistance: func(t *testing.T, r *Router, s *app.Session) string {
		ids := nPoints3D(1)(t, r, s)
		return fmt.Sprintf(`{"sketchIndex":0,"kind":"pointPlaneDistance","entities":[%d],"plane":"XY","expression":"5 mm"}`, ids[0])
	},
	types.Dim3DTwoLineAngle: func(t *testing.T, r *Router, s *app.Session) string {
		ids := twoLines3D(t, r, s)
		return fmt.Sprintf(`{"sketchIndex":0,"kind":"twoLineAngle","entities":[%d,%d],"expression":"45 deg"}`, ids[0], ids[1])
	},
	types.Dim3DSplineLength: func(t *testing.T, r *Router, s *app.Session) string {
		var sp wire.AddSketch3DEntityResult
		call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,0,0],[1,1,0],[2,0,1]]}`, &sp)
		return fmt.Sprintf(`{"sketchIndex":0,"kind":"splineLength","entities":[%d],"expression":"40 mm"}`, sp.EntityID)
	},
}

// trackedUnimplemented3DDimensionKinds mirrors the constraint guard's escape hatch —
// currently empty: every declared dimension kind is implemented.
var trackedUnimplemented3DDimensionKinds = map[types.Dimension3DConstraintKind]string{}

// allDeclared3DDimensionKinds mirrors the const block in api/types/sketch3d.go; keep
// in sync when the API grows.
var allDeclared3DDimensionKinds = []types.Dimension3DConstraintKind{
	types.Dim3DDistance, types.Dim3DLineLength, types.Dim3DRadius,
	types.Dim3DPointPlaneDistance, types.Dim3DTwoLineAngle, types.Dim3DSplineLength,
}

// TestEvery3DDimensionKindAccepted drives sketch3d.addDimension for every declared
// kind and asserts it lands and enumerates with the same kind.
func TestEvery3DDimensionKindAccepted(t *testing.T) {
	for kind, fixture := range declared3DDimensionFixtures {
		t.Run(string(kind), func(t *testing.T) {
			r, s := emptyPartSession(t)
			call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
			var res wire.AddSketch3DDimensionResult
			call(t, r, s, "sketch3d.addDimension", fixture(t, r, s), &res)
			if res.Kind != string(kind) {
				t.Errorf("addDimension kind = %q, want %q", res.Kind, kind)
			}
			var dims wire.ListDimensions3DResult
			call(t, r, s, "sketch3d.dimensions", `{"sketchIndex":0}`, &dims)
			if len(dims.Dimensions) != 1 || dims.Dimensions[0].Kind != string(kind) {
				t.Errorf("enumerated dimensions = %+v, want one of kind %q", dims.Dimensions, kind)
			}
		})
	}
}

// TestDeclared3DDimensionKindsComplete fails when a declared kind has neither a
// fixture nor a tracking issue — api/types cannot ship dimensions ahead of the
// solver (issue #144's lasting guard).
func TestDeclared3DDimensionKindsComplete(t *testing.T) {
	for _, kind := range allDeclared3DDimensionKinds {
		_, covered := declared3DDimensionFixtures[kind]
		issue, tracked := trackedUnimplemented3DDimensionKinds[kind]
		switch {
		case covered && tracked:
			t.Errorf("kind %q is both covered and tracked-unimplemented (%s) — remove one", kind, issue)
		case !covered && !tracked:
			t.Errorf("kind %q is declared in api/types but has no fixture and no tracking issue", kind)
		}
	}
}
