// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// emptyAssemblySession returns a router and a session with a fresh assembly active — its origin
// frame is seeded, the fixture for the assembly work-feature wire tests.
func emptyAssemblySession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	s := app.NewSession()
	d, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(d); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	return New(opregistry.Default()), s
}

// TestWorkPlanesListServesAssembly: an assembly answers workPlanes.list with its three origin
// planes — before, the handler rejected an assembly ("not a part").
func TestWorkPlanesListServesAssembly(t *testing.T) {
	t.Parallel()
	r, s := emptyAssemblySession(t)
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	if len(list.Planes) != 3 {
		t.Fatalf("fresh assembly has %d work planes, want 3 origin planes", len(list.Planes))
	}
	for _, p := range list.Planes {
		if !p.IsOrigin || !p.Healthy {
			t.Errorf("origin plane %q: isOrigin=%v healthy=%v, want both true", p.Name, p.IsOrigin, p.Healthy)
		}
	}
}

// TestWorkPlanesCreateOffsetInAssembly: an offset plane built on the assembly's origin XY plane is
// created, healthy, and lands at the expected position — the same flow a part supports (its offset
// plane needs no body), now served for an assembly. Shares the assertion with the part test.
func TestWorkPlanesCreateOffsetInAssembly(t *testing.T) {
	t.Parallel()
	r, s := emptyAssemblySession(t)
	assertOffsetPlaneOnXY(t, r, s)
}

// TestWorkPlanesCreateThreePointsInAssembly: a three-point plane through the assembly's origin
// datums is created and healthy — exercising a reference-only constructor against an assembly.
func TestWorkPlanesCreateThreePointsInAssembly(t *testing.T) {
	t.Parallel()
	r, s := emptyAssemblySession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"two-planes","refs":["origin/plane/xy","origin/plane/xz"]}`, &res)
	if !res.Healthy {
		t.Errorf("assembly midplane not healthy: %+v", res)
	}
}

// TestWorkPointCreateInAssembly: a datum point fixed at a position is added to an assembly and
// returns a usable reference.
func TestWorkPointCreateInAssembly(t *testing.T) {
	t.Parallel()
	r, s := emptyAssemblySession(t)
	var res wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[1,2,3]}`, &res)
	if res.Ref == "" {
		t.Errorf("workPoints.create in assembly returned no ref: %+v", res)
	}
}
