// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func geomRefBox() *topo.Body { return subd.ToBody(subd.Box(4, 4, 4), "box") }

// TestGeometricEdgeFilletBindsAndRounds proves the M8/ADR-0040 path: a fillet whose edge
// is given only by a GEOMETRIC descriptor (no Oblikovati lineage key — what the NX exporter
// can supply) binds to the running body's edge and actually rounds it, reducing volume.
func TestGeometricEdgeFilletBindsAndRounds(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	box := geomRefBox()
	NewBaseFeatures(fs).AddBase(box)

	ref := topo.DescribeEdge(box.Edges()[0])
	pf := NewDressUpFeatures(fs).addFillet(&FilletDefinition{
		GeomEdges: []topo.GeometricEdgeRef{ref},
		Radius:    constFloat(0.5),
	})
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("geometric-edge fillet is not healthy: %+v", pf.Health())
	}
	res := fs.Result()
	if len(res) != 1 {
		t.Fatalf("result has %d bodies, want 1", len(res))
	}
	boxVol := query.BodyGeometryProperties(geomRefBox(), ops.DefaultQuality()).Volume
	gotVol := query.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume
	if !(gotVol < boxVol) {
		t.Errorf("fillet did not reduce volume: got %g, box %g", gotVol, boxVol)
	}
}

// TestGeometricEdgeFilletMissGoesSick confirms a descriptor that binds nothing fails the
// feature honestly rather than silently dropping the selection.
func TestGeometricEdgeFilletMissGoesSick(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(geomRefBox())
	far := topo.GeometricEdgeRef{Midpoint: math.P3(100, 100, 100)}
	pf := NewDressUpFeatures(fs).addFillet(&FilletDefinition{
		GeomEdges: []topo.GeometricEdgeRef{far},
		Radius:    constFloat(0.5),
	})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("a fillet whose geometric edge binds nothing should not be healthy")
	}
}

// TestGeometricEdgeRefSurvivesRecipeRoundTrip checks the geomEdges encoding round-trips
// through serialize → restore (the exporter writes this; the reader must restore it).
func TestGeometricEdgeRefSurvivesRecipeRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	ref := topo.DescribeEdge(geomRefBox().Edges()[0])
	pf := NewDressUpFeatures(fs).addFillet(&FilletDefinition{
		GeomEdges: []topo.GeometricEdgeRef{ref},
		Radius:    constFloat(0.5),
	})

	fd, err := serializeFeature(pf, nil, map[ID]int{})
	if err != nil {
		t.Fatalf("serializeFeature: %v", err)
	}
	if fd.Fillet == nil || len(fd.Fillet.GeomEdges) != 1 {
		t.Fatalf("recipe did not carry the geometric edge ref: %+v", fd.Fillet)
	}

	dst := NewPartFeatures(nil)
	rpf, err := buildFeature(dst, fd, nil, nil, nil)
	if err != nil {
		t.Fatalf("buildFeature: %v", err)
	}
	ff, ok := rpf.feature.(*FilletFeature)
	if !ok {
		t.Fatalf("restored feature is %T, want *FilletFeature", rpf.feature)
	}
	if len(ff.def.GeomEdges) != 1 {
		t.Fatalf("restored fillet has %d geom edges, want 1", len(ff.def.GeomEdges))
	}
	if ff.def.GeomEdges[0].Midpoint.DistanceTo(ref.Midpoint) > 1e-9 {
		t.Errorf("restored midpoint %v != original %v", ff.def.GeomEdges[0].Midpoint, ref.Midpoint)
	}
}
