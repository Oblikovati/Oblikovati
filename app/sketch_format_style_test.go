// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// The lists set the format of the selected geometry, and the three compose rather than overwrite
// one another.
func TestSetSelectionFormatComposes(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})

	if n := s.SetSelectionLineType("dashed"); n != 1 {
		t.Fatalf("styled = %d, want 1", n)
	}
	s.SetSelectionColor(types.NewColor(255, 0, 0))
	s.SetSelectionLineWeight(0.35)

	f, ok := sk.EntityFormat(l.EntityID())
	if !ok {
		t.Fatal("the entity must carry an override")
	}
	if f.LineType != "dashed" || !f.Color.IsOverride() || f.LineWeight != 0.35 {
		t.Errorf("format = %+v, want all three fields set", f)
	}
}

// Setting a field back to Default clears just that field, leaving the others.
func TestSetSelectionFormatToDefault(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})
	s.SetSelectionLineType("dashed")
	s.SetSelectionLineWeight(0.35)

	s.SetSelectionLineType("")
	f, ok := sk.EntityFormat(l.EntityID())
	if !ok {
		t.Fatal("the entity still overrides its weight, so an entry must remain")
	}
	if f.LineType != "" || f.LineWeight != 0.35 {
		t.Errorf("format = %+v, want the line type cleared and the weight kept", f)
	}
}

// Clearing the last override removes the entry entirely — absence is the single representation
// of Default.
func TestClearingEveryFieldRemovesTheEntry(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})
	s.SetSelectionLineType("dashed")
	s.SetSelectionLineType("")
	if n := sk.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}

// A mixed selection reads as Default in the lists where its entities disagree, so a list never
// claims a value the selection does not uniformly have.
func TestSelectionFormatOfAMixedSelection(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	a := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	b := sk.Lines().AddByTwoPoints(math.P2(0, 5), math.P2(10, 5))
	s.Select(SketchEntityHandle{Entity: a})
	s.SetSelectionLineType("dashed")
	s.SetSelectionLineWeight(0.35)

	s.Selection().Clear()
	s.Select(SketchEntityHandle{Entity: b})
	s.SetSelectionLineWeight(0.35) // b shares the weight but not the line type

	s.Selection().Clear()
	s.Select(SketchEntityHandle{Entity: a})
	s.Select(SketchEntityHandle{Entity: b})
	f := s.SelectionFormat()
	if f.LineType != "" {
		t.Errorf("line type = %q, want Default — the selection disagrees", f.LineType)
	}
	if f.LineWeight != 0.35 {
		t.Errorf("weight = %v, want 0.35 — the selection agrees", f.LineWeight)
	}
}

// Every list starts with Default, and choosing it clears that field.
func TestFormatListsStartWithDefault(t *testing.T) {
	t.Parallel()
	for _, kind := range []FormatListKind{LineTypeList, ColorList, LineWeightList} {
		entries := FormatListEntries(kind)
		if len(entries) < 2 {
			t.Fatalf("kind %d has %d entries, want Default plus values", kind, len(entries))
		}
		if entries[0].Label != "Default" {
			t.Errorf("kind %d entry 0 = %q, want Default", kind, entries[0].Label)
		}
	}
}

// The list reports which row the selection sits on, so the ribbon shows the current value.
func TestFormatListSelectionTracksTheEntity(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})

	if got := s.FormatListSelection(LineTypeList); got != 0 {
		t.Errorf("unstyled selection = row %d, want 0 (Default)", got)
	}
	s.ChooseFormatListEntry(LineTypeList, 2)
	if got := s.FormatListSelection(LineTypeList); got != 2 {
		t.Errorf("after choosing row 2 the list reads row %d", got)
	}
	s.ChooseFormatListEntry(LineTypeList, 0) // back to Default
	if got := s.FormatListSelection(LineTypeList); got != 0 {
		t.Errorf("after choosing Default the list reads row %d, want 0", got)
	}
}

// An out-of-range row changes nothing rather than panicking.
func TestChooseFormatListEntryRejectsBadRow(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})
	if n := s.ChooseFormatListEntry(LineTypeList, 99); n != 0 {
		t.Errorf("styled = %d, want 0 for an out-of-range row", n)
	}
	if sk.EntityFormatCount() != 0 {
		t.Error("an out-of-range row must store nothing")
	}
}
