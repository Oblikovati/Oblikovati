// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

// tornBodyFeature is the named fake kernel op for the post-condition: it "succeeds" — no error, no
// panic — and hands back a body DECLARED solid whose single face leaves four boundary edges. That is
// what a boolean which dropped a face produces, and before the post-condition the engine stored it,
// reported health OK and meshed it.
type tornBodyFeature struct{ built *topo.Body }

func (f *tornBodyFeature) Kind() string { return "tornfake" }

func (f *tornBodyFeature) Recompute(in Input) (Output, error) {
	f.built = tornSquareSolid()
	return Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), f.built)}, nil
}

// tornSquareSolid builds one planar face declared SOLID: four boundary edges, closed=false.
func tornSquareSolid() *topo.Body {
	lin := topo.NewLineage(topo.Tok("test", "torn", 0))
	bld := topo.NewBuilder(true, lin)
	corners := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}
	verts := make([]*topo.Vertex, len(corners))
	for i, p := range corners {
		verts[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, len(corners))
	for i := range corners {
		j := (i + 1) % len(corners)
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), verts[i], verts[j], lin))
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, lin, topo.OuterLoop(uses...))
	return bld.Build()
}

// TestAnInvalidResultBodySickensItsFeature is the post-condition's regression. Before it, the fake
// above produced: health = ok, diagnostics = none, one body in the result.
func TestAnInvalidResultBodySickensItsFeature(t *testing.T) {
	t.Parallel()
	fake := &tornBodyFeature{}
	fs := NewPartFeatures(nil)
	pf := fs.Add(fake)
	fs.Recompute()

	if pf.Health().Status != health.Sick {
		t.Fatalf("a feature that produced an invalid body must be sick; got %+v", pf.Health())
	}
	if !hasDiagCode(pf.Diagnostics(), CodeFeatureInvalidResult) {
		t.Errorf("the post-condition must record a named diagnostic; got %v", pf.Diagnostics())
	}
	if got := len(fs.Result()); got != 0 {
		t.Errorf("the invalid body must be DROPPED, not stored; the result holds %d bodies", got)
	}
}

// The reason the browser shows must name the invariant that broke, not "recompute failed".
func TestThePostconditionReasonNamesTheInvariant(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	pf := fs.Add(&tornBodyFeature{})
	fs.Recompute()
	for _, want := range []string{"tornfake", "closed=false", "boundary (open) edge"} {
		if !strings.Contains(pf.Health().Reason, want) {
			t.Errorf("the sick reason must name %q; got %q", want, pf.Health().Reason)
		}
	}
}

// Failure is LOCAL: the sick feature quarantines its dependents and the rebuild continues.
func TestAnInvalidResultQuarantinesTheDependents(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	bad := fs.Add(&tornBodyFeature{})
	downstream := fs.Add(passThroughFeature{}, bad.ID())
	fs.Recompute()
	if bad.Health().Status != health.Sick {
		t.Fatalf("the producer must be sick; got %+v", bad.Health())
	}
	if downstream.Health().Status != health.Sick {
		t.Errorf("a dependent of a sick feature must be quarantined; got %+v", downstream.Health())
	}
}

// postconditionError measures what the feature BUILT. A body it passed through unchanged was already
// validated at the exit of the feature that built it, so re-checking it would make the post-condition
// cost O(features x bodies) for an answer that cannot have changed.
func TestThePostconditionSkipsPassedThroughBodies(t *testing.T) {
	t.Parallel()
	torn := tornSquareSolid()
	in := []*topo.Body{torn}
	if err := postconditionError(passThroughFeature{}, in, in, nil); err != nil {
		t.Errorf("a body the feature did not build must not be re-validated: %v", err)
	}
	if err := postconditionError(&tornBodyFeature{}, nil, in, nil); !errors.Is(err, ErrInvalidFeatureResult) {
		t.Errorf("the same body BUILT here must fail the post-condition; got %v", err)
	}
}

// An ADOPTED body — an imported STEP, a derived component's source — is REPORTED, not refused: its
// defect belongs to the file, and refusing every imperfect import is a product decision. Measured on
// the blend-parity corpus's own fixtures: simple/H3 and simple/H5 import with three boundary edges.
func TestAnAdoptedInvalidBodyIsReportedNotRefused(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	pf := NewBaseFeatures(fs).AddBase(tornSquareSolid())
	fs.Recompute()

	if pf.Health().Status == health.Sick {
		t.Errorf("an imported body must not sicken the feature that wraps it; got %+v", pf.Health())
	}
	if !hasDiagCode(pf.Diagnostics(), CodeFeatureInvalidResult) {
		t.Errorf("an invalid imported body must still be REPORTED; got %v", pf.Diagnostics())
	}
	if got := len(fs.Result()); got != 1 {
		t.Errorf("an adopted body is kept, not dropped; the result holds %d bodies", got)
	}
}

// A valid result stays silent: the post-condition is a gate, not a new diagnostic on every feature.
func TestAValidResultKeepsItsFeatureHealthy(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	extrudes := NewExtrudeFeatures(fs)
	base := extrudes.AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 2 })
	fs.Recompute()
	if !base.Health().OK() {
		t.Fatalf("a plain extrude must stay healthy: %+v", base.Health())
	}
	if hasDiagCode(base.Diagnostics(), CodeFeatureInvalidResult) {
		t.Errorf("a valid result must record no post-condition defect; got %v", base.Diagnostics())
	}
}
