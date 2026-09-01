// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestCreateEdgeAxes dispatches analytic-edge and line-by-entity axes over the wire. With no body
// the edge reference does not resolve, so each is created but reports healthy=false — exercising the
// edge dispatch (#1840).
func TestCreateEdgeAxes(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	for _, kind := range []string{"analytic-edge", "line-by-entity"} {
		var res wire.CreateWorkAxisResult
		call(t, r, s, "workAxes.create", `{"kind":"`+kind+`","refs":["edge/AAAA"]}`, &res)
		if res.Ref == "" {
			t.Errorf("%s axis should be created even when its edge is unresolved", kind)
		}
		if res.Healthy {
			t.Errorf("%s: with no body the edge is unresolved, so healthy should be false", kind)
		}
	}
}

// TestCreateEdgeAxisGeometricRef dispatches an analytic-edge axis whose reference is an ADR-0040
// geometric descriptor string — the external-author form routes through the same create (#1840).
func TestCreateEdgeAxisGeometricRef(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	ref := types.GeometricEdgeRef{Midpoint: [3]float64{1, 2, 3}, Direction: [3]float64{1, 0, 0}}.Ref()
	var res wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create", `{"kind":"analytic-edge","refs":["`+ref+`"]}`, &res)
	if res.Ref == "" {
		t.Error("a geometric-edge-ref axis should be created (unresolved without a body)")
	}
}

// TestCreateEdgeMidpointPoint dispatches an edge-midpoint point over the wire (#1842).
func TestCreateEdgeMidpointPoint(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"kind":"edge-midpoint","refs":["edge/AAAA"]}`, &res)
	if res.Ref == "" || res.Healthy {
		t.Errorf("edge-midpoint should be created but unresolved (healthy=false): %+v", res)
	}
}

// TestCreateEdgeAxisWrongRefCount: an edge axis needs exactly one edge reference (#1840).
func TestCreateEdgeAxisWrongRefCount(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workAxes.create", []byte(`{"kind":"analytic-edge","refs":["a","b"]}`)); err == nil {
		t.Error("analytic-edge with two references should error")
	}
	if _, err := r.Handle(s, "workPoints.create", []byte(`{"kind":"edge-midpoint"}`)); err == nil {
		t.Error("edge-midpoint with no reference should error")
	}
}
