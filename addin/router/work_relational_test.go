// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// Relational work-axis (#1840) and work-point (#1842) constructors over the wire, plus
// workPoints.list enumeration.

func TestWorkAxisRelationalKinds(t *testing.T) {
	r, s := emptyPartSession(t)
	var pt wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[0,0,5]}`, &pt)

	cases := []struct{ name, args string }{
		{"point-and-plane", `{"kind":"point-and-plane","refs":["` + pt.Ref + `","origin/plane/xy"]}`},
		{"line-and-point", `{"kind":"line-and-point","refs":["origin/axis/x","` + pt.Ref + `"]}`},
		{"line-and-plane", `{"kind":"line-and-plane","refs":["origin/axis/x","origin/plane/xy"]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var res wire.CreateWorkAxisResult
			call(t, r, s, "workAxes.create", c.args, &res)
			if !res.Healthy {
				t.Errorf("%s axis not healthy: %+v", c.name, res)
			}
		})
	}
}

func TestWorkPointRelationalKinds(t *testing.T) {
	r, s := emptyPartSession(t)
	var src wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[1,2,3]}`, &src)

	cases := []struct{ name, args string }{
		{"point", `{"kind":"point","refs":["` + src.Ref + `"]}`},
		{"two-lines", `{"kind":"two-lines","refs":["origin/axis/x","origin/axis/y"]}`},
		{"three-planes", `{"kind":"three-planes","refs":["origin/plane/xy","origin/plane/xz","origin/plane/yz"]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var res wire.CreateWorkPointResult
			call(t, r, s, "workPoints.create", c.args, &res)
			if !res.Healthy {
				t.Errorf("%s point not healthy: %+v", c.name, res)
			}
		})
	}
}

func TestWorkPointRelationalWrongRefCount(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPoints.create", []byte(`{"kind":"three-planes","refs":["origin/plane/xy"]}`)); err == nil {
		t.Error("three-planes with one ref should error")
	}
}

// TestWorkPointsList enumerates points and confirms position/kind/origin read-back (#1842).
func TestWorkPointsList(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "workPoints.create", `{"at":[1,2,3]}`, &wire.CreateWorkPointResult{})
	call(t, r, s, "workPoints.create", `{"kind":"two-lines","refs":["origin/axis/x","origin/axis/y"]}`, &wire.CreateWorkPointResult{})

	var list wire.ListWorkPointsResult
	call(t, r, s, "workPoints.list", "{}", &list)
	// origin centre + 2 user points.
	if len(list.Points) < 3 {
		t.Fatalf("listed %d points, want at least 3 (origin + 2)", len(list.Points))
	}
	var origin, twoLines *wire.WorkPointInfo
	for i := range list.Points {
		p := &list.Points[i]
		if p.IsOrigin {
			origin = p
		}
		if p.Kind == "two-lines" {
			twoLines = p
		}
	}
	if origin == nil {
		t.Error("list should include the origin centre point")
	}
	if twoLines == nil {
		t.Fatal("list should include the two-lines point")
	}
	if p := twoLines.Position; len(p) != 3 || p[0] != 0 || p[1] != 0 || p[2] != 0 {
		t.Errorf("two-lines point position = %v, want the origin (0,0,0)", twoLines.Position)
	}
}
