// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/addin/modelaccess"
)

// TestModelLengthClosureStaysLiveForParamExpr is the regression guard for issue #1230: a distance
// argument that references a parameter (e.g. a work-plane offset "h" or a sketch-pattern spacing
// "L - 2*m") must stay LIVE, so editing the parameter re-evaluates it on the next recompute. The
// earlier fast path baked any expression that evaluated at creation time — including parameter
// expressions — which froze work-plane offsets and pattern spacings (the NopSCAD loft + pcb
// failures). A pure literal has no parameter dependency and is baked.
func TestModelLengthClosureStaysLiveForParamExpr(t *testing.T) {
	_, s := emptyPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	if _, err := part.Parameters().AddUserParameter("h", "10 mm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}

	// A parameter expression: the closure must track an edit to the parameter.
	live, err := modelLengthClosure(part, "h")
	if err != nil {
		t.Fatalf("modelLengthClosure(h): %v", err)
	}
	before := live()
	if before <= 0 {
		t.Fatalf("offset at h=10mm = %v, want > 0", before)
	}
	p, _ := part.Parameters().ByName("h")
	if err := part.Parameters().SetExpression(p.ID(), "25 mm"); err != nil {
		t.Fatalf("set h: %v", err)
	}
	part.Recompute()
	after := live()
	if math.Abs(after-2.5*before) > 1e-9 {
		t.Errorf("after h 10mm→25mm offset = %v, want %v (2.5×) — the parameter expression was frozen", after, 2.5*before)
	}

	// A pure literal has no parameter dependency: it is a constant value.
	lit, err := modelLengthClosure(part, "5 mm")
	if err != nil {
		t.Fatalf("modelLengthClosure(5 mm): %v", err)
	}
	if got := lit(); math.Abs(got-0.5) > 1e-9 { // database unit is the centimetre (5 mm = 0.5 cm)
		t.Errorf("literal 5 mm = %v, want 0.5 (cm)", got)
	}
}
