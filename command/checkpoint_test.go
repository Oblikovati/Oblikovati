// SPDX-License-Identifier: GPL-2.0-only

package command

import "testing"

func TestGoToCheckPointRestoresCapturedState(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	d.SetDisplayName("v0")

	tx := h.Begin("to v1")
	_ = tx.Do(Rename(d, "v1"))
	_ = tx.Commit()

	cp := h.SetCheckPoint("after v1") // capture at depth 1, name == "v1"

	// More edits move the model forward.
	for _, n := range []string{"v2", "v3"} {
		tx := h.Begin("to " + n)
		_ = tx.Do(Rename(d, n))
		_ = tx.Commit()
	}
	if d.DisplayName() != "v3" {
		t.Fatalf("name = %q, want v3 before checkpoint return", d.DisplayName())
	}

	if err := h.GoToCheckPoint(cp); err != nil {
		t.Fatalf("GoToCheckPoint: %v", err)
	}
	if d.DisplayName() != "v1" {
		t.Errorf("after GoToCheckPoint name = %q, want v1", d.DisplayName())
	}

	// And forward again to a checkpoint at the latest depth.
	end := CheckPoint{label: "end", depth: 3}
	if err := h.GoToCheckPoint(end); err != nil {
		t.Fatalf("GoToCheckPoint forward: %v", err)
	}
	if d.DisplayName() != "v3" {
		t.Errorf("after forward GoToCheckPoint name = %q, want v3", d.DisplayName())
	}
}

func TestCheckPointAccessorsAndOneUpdate(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	tx := h.Begin("e1")
	_ = tx.Do(Rename(d, "e1"))
	_ = tx.Commit()

	cp := h.SetCheckPoint("mark")
	if cp.Label() != "mark" || cp.Depth() != 1 {
		t.Errorf("checkpoint = %+v, want {mark 1}", cp)
	}
	tx2 := h.Begin("e2")
	_ = tx2.Do(Rename(d, "e2"))
	_ = tx2.Commit()

	updates := 0
	h.OnChange(func() { updates++ })
	if err := h.GoToCheckPoint(cp); err != nil { // undoes one step
		t.Fatalf("GoToCheckPoint: %v", err)
	}
	if updates != 1 {
		t.Errorf("GoToCheckPoint fired %d updates, want 1 coalesced", updates)
	}

	if len(h.CheckPoints()) != 1 {
		t.Errorf("CheckPoints len = %d, want 1", len(h.CheckPoints()))
	}
	if !h.ReleaseCheckPoint(cp) || len(h.CheckPoints()) != 0 {
		t.Error("ReleaseCheckPoint did not remove the checkpoint")
	}
	if h.ReleaseCheckPoint(cp) {
		t.Error("ReleaseCheckPoint returned true for an unknown checkpoint")
	}
}

func TestGoToCheckPointGuards(t *testing.T) {
	h := NewHistory()
	open := h.Begin("open")
	if err := h.GoToCheckPoint(CheckPoint{depth: 0}); err == nil {
		t.Error("GoToCheckPoint allowed with an open transaction")
	}
	_ = open.Abort()
	// A depth that can never be reached (no steps to redo up to it) errors.
	if err := h.GoToCheckPoint(CheckPoint{label: "future", depth: 5}); err == nil {
		t.Error("GoToCheckPoint to an unreachable depth did not error")
	}
}
