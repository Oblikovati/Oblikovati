// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

// mutableBodyResolver is a named fake whose body can be swapped to simulate a part recompute
// (which rebuilds the body, changing its pointer).
type mutableBodyResolver struct{ body *topo.Body }

func (m *mutableBodyResolver) Body(string) (*topo.Body, bool) { return m.body, m.body != nil }

// TestSyncViewsTracksModelRecompute is the live-associativity guarantee: after the referenced
// model is recomputed (a new body), SyncViews re-projects the views to the new geometry, while
// an unchanged model is a no-op (cheap). A 2×2×2 cube → a 2×2×8 box changes the iso view's
// extent, proving the curves were re-projected.
func TestSyncViewsTracksModelRecompute(t *testing.T) {
	resolver := &mutableBodyResolver{body: subd.ToBody(subd.Box(2, 2, 2), "box")}
	c := NewContent()
	c.SetBodyResolver(resolver)
	c.SetModelReference("box.opd")

	v, err := c.Sheets().Active().Views().AddBase(BaseViewSpec{Orientation: types.BaseViewIso, Scale: 1})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	_, _, _, maxY0, _ := v.BoundsMM()

	// An unchanged model: SyncViews must not re-project (same body pointer).
	beforeCurves := v.CurveCount()
	c.SyncViews()
	if v.CurveCount() != beforeCurves {
		t.Errorf("SyncViews re-projected an unchanged model (%d→%d curves)", beforeCurves, v.CurveCount())
	}

	// Recompute the part: swap in a taller box (new body pointer), then SyncViews.
	resolver.body = subd.ToBody(subd.Box(2, 2, 8), "box")
	c.SyncViews()
	_, _, _, maxY1, _ := v.BoundsMM()
	if maxY1 <= maxY0 {
		t.Fatalf("iso view did not grow after the model got taller: maxY %g → %g", maxY0, maxY1)
	}
}
