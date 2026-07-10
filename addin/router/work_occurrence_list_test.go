// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// TestOccurrenceDatumListAndInput drives #1857 over the wire: a placed sub-component's datum plane is
// surfaced by workPlanes.list (occurrence path) as an occurrence-qualified ref with its geometry
// transformed into the assembly, and that ref is then accepted as a plane input to an assembly
// work-plane create (AC1–AC3).
func TestOccurrenceDatumListAndInput(t *testing.T) {
	r, s := emptyAssemblySession(t)
	asm := s.ActiveDocument().Content().(*compdef.AssemblyComponentDefinition)
	part := compdef.NewPartComponentDefinition()
	part.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 5 }) // z = 5 in part space
	zdir, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	part.WorkAxes().AddByLine(math.P3(0, 0, 0), zdir)
	part.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 2, 3) })
	part.Recompute()
	asm.Place("sub:1", part, math.Translation4(math.V3(10, 0, 0))) // placed +10 along X

	// Axes and points also list as occurrence-qualified refs, transformed.
	var axes wire.ListWorkAxesResult
	call(t, r, s, "workAxes.list", `{"occurrence":["sub:1"]}`, &axes)
	foundAxis := false
	for _, a := range axes.Axes {
		if strings.HasPrefix(a.Ref, "occ/") && !a.IsOrigin {
			foundAxis = true
		}
	}
	if !foundAxis {
		t.Error("expected an occurrence-qualified axis in the assembly-context list")
	}
	var points wire.ListWorkPointsResult
	call(t, r, s, "workPoints.list", `{"occurrence":["sub:1"]}`, &points)
	for _, p := range points.Points {
		if strings.HasPrefix(p.Ref, "occ/") && !p.IsOrigin {
			if stdmath.Abs(p.Position[0]-11) > 1e-9 { // (1,2,3) placed +10 X → (11,2,3)
				t.Errorf("occurrence point position = %v, want x=11", p.Position)
			}
		}
	}

	// AC1+AC2: list the sub-component's datum planes as occurrence-qualified refs.
	var planes wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", `{"occurrence":["sub:1"]}`, &planes)
	occRef := ""
	for _, p := range planes.Planes {
		if p.IsOrigin || !strings.HasPrefix(p.Ref, "occ/") {
			continue
		}
		occRef = p.Ref
		if len(p.Origin) != 3 || stdmath.Abs(p.Origin[0]-10) > 1e-9 || stdmath.Abs(p.Origin[2]-5) > 1e-9 {
			t.Errorf("occurrence plane origin = %v, want x=10 z=5 (part z=5 placed +10 X)", p.Origin)
		}
	}
	if occRef == "" {
		t.Fatal("expected an occurrence-qualified user plane in the assembly-context list")
	}

	// AC3: the occurrence-qualified ref is accepted as a plane input to an assembly work-plane create.
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", fmt.Sprintf(`{"kind":"plane-offset","refs":["%s"],"offset":"2 mm"}`, occRef), &res)
	if !res.Healthy {
		t.Errorf("an assembly work plane offset from an occurrence plane should be healthy: %+v", res)
	}
}

// TestOccurrenceListErrors: an occurrence path on a part (no occurrences), and an unknown path on an
// assembly, both error cleanly (#1857).
func TestOccurrenceListErrors(t *testing.T) {
	rp, sp := emptyPartSession(t)
	if _, err := rp.Handle(sp, "workPlanes.list", []byte(`{"occurrence":["sub:1"]}`)); err == nil {
		t.Error("an occurrence path on a part should error (no occurrences)")
	}
	ra, sa := emptyAssemblySession(t)
	if _, err := ra.Handle(sa, "workAxes.list", []byte(`{"occurrence":["missing:1"]}`)); err == nil {
		t.Error("an unknown occurrence path should error")
	}
}
