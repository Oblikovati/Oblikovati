// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "testing"

func TestPartHasThreeOriginPlanes(t *testing.T) {
	d := NewPartComponentDefinition()
	planes := d.OriginPlanes()
	if len(planes) != 3 {
		t.Fatalf("part has %d origin planes, want 3", len(planes))
	}
	want := []string{"XY Plane", "XZ Plane", "YZ Plane"}
	for i, n := range want {
		if planes[i].Name() != n {
			t.Errorf("origin plane %d = %q, want %q", i, planes[i].Name(), n)
		}
		if planes[i].DisplaySize() <= 0 {
			t.Errorf("%s has non-positive display size", n)
		}
	}
}

func TestWorkPlaneByName(t *testing.T) {
	d := NewPartComponentDefinition()
	if wp, ok := d.WorkPlaneByName("XZ Plane"); !ok || wp.Name() != "XZ Plane" {
		t.Errorf("WorkPlaneByName(XZ) = %v/%v, want the XZ plane", wp, ok)
	}
	if _, ok := d.WorkPlaneByName("nope"); ok {
		t.Error("WorkPlaneByName found a non-existent plane")
	}
}

func TestOriginPlanesAreDistinctOrientations(t *testing.T) {
	d := NewPartComponentDefinition()
	p := d.OriginPlanes()
	// The three planes' normals must be mutually distinct (independent axes).
	nxy := p[0].Plane().Normal().AsVector()
	nxz := p[1].Plane().Normal().AsVector()
	nyz := p[2].Plane().Normal().AsVector()
	if nxy.IsEqualTo(nxz, 1e-9) || nxy.IsEqualTo(nyz, 1e-9) || nxz.IsEqualTo(nyz, 1e-9) {
		t.Errorf("origin plane normals are not distinct: %v %v %v", nxy, nxz, nyz)
	}
}
