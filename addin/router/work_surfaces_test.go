// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// patchedSurfacePartViaAPI drives the wire surface to a part with one boundary-patch
// surface body (a closed rectangle filled with a sheet), the fixture the WorkSurface
// methods read (M20-F16).
func patchedSurfacePartViaAPI(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"boundaryPatch","args":{"sketchIndex":0}}`, &struct{}{})
	return r, s
}

func TestWorkSurfacesListReportsPatchSurface(t *testing.T) {
	t.Parallel()
	r, s := patchedSurfacePartViaAPI(t)
	var got wire.ListWorkSurfacesResult
	call(t, r, s, "workSurfaces.list", `{}`, &got)
	if len(got.Surfaces) != 1 {
		t.Fatalf("workSurfaces.list = %d surfaces, want 1", len(got.Surfaces))
	}
	su := got.Surfaces[0]
	if su.Name != "Surface1" || su.Ref != "surface/0" || !su.Visible || su.Bodies != 1 {
		t.Errorf("surface = %+v, want Surface1/surface/0/visible/1 body", su)
	}
}

func TestWorkSurfacesSetVisibleAndRename(t *testing.T) {
	t.Parallel()
	r, s := patchedSurfacePartViaAPI(t)

	var hidden wire.WorkSurfaceDetailResult
	call(t, r, s, "workSurfaces.setVisible", mustJSON(t, wire.SetWorkSurfaceVisibleArgs{Index: 0, Visible: false}), &hidden)
	if hidden.Surface.Visible {
		t.Errorf("after setVisible(false) surface is still visible: %+v", hidden.Surface)
	}

	var renamed wire.WorkSurfaceDetailResult
	call(t, r, s, "workSurfaces.rename", mustJSON(t, wire.RenameWorkSurfaceArgs{Index: 0, Name: "Parting"}), &renamed)
	if renamed.Surface.Name != "Parting" {
		t.Errorf("after rename, name = %q, want Parting", renamed.Surface.Name)
	}

	// The change is observable via get and survives (state held in the collection).
	var got wire.WorkSurfaceDetailResult
	call(t, r, s, "workSurfaces.get", mustJSON(t, wire.WorkSurfaceRefArgs{Index: 0}), &got)
	if got.Surface.Name != "Parting" || got.Surface.Visible {
		t.Errorf("get = %+v, want hidden surface named Parting", got.Surface)
	}
}

func TestWorkSurfacesRenameRejectsEmpty(t *testing.T) {
	t.Parallel()
	r, s := patchedSurfacePartViaAPI(t)
	if _, err := r.Handle(s, "workSurfaces.rename", []byte(mustJSON(t, wire.RenameWorkSurfaceArgs{Index: 0, Name: ""}))); err == nil {
		t.Error("empty rename must be rejected")
	}
}

func TestWorkSurfacesGetOutOfRangeFails(t *testing.T) {
	t.Parallel()
	r, s := patchedSurfacePartViaAPI(t)
	if _, err := r.Handle(s, "workSurfaces.get", []byte(mustJSON(t, wire.WorkSurfaceRefArgs{Index: 9}))); err == nil {
		t.Error("get with an out-of-range index must fail")
	}
}
