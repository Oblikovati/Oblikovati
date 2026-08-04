// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Show Format's documented behaviour is the inverse of its name: on suppresses the overrides and
// draws with default attributes; off shows user formatting again.
func TestShowFormatSuppressesOverrides(t *testing.T) {
	_, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	sk.SetEntityFormat(l.EntityID(), sketch.EntityFormat{
		LineType: "dashed", Color: types.NewColor(255, 0, 0), LineWeight: 0.5,
	})

	styled := SketchEntityStyle(sk, l, false)
	if !styled.Color.IsOverride() || styled.LineWeight != 0.5 {
		t.Errorf("with Show Format off the override must apply, got %+v", styled)
	}

	suppressed := SketchEntityStyle(sk, l, true)
	if suppressed.Color.IsOverride() || suppressed.LineWeight != 0 {
		t.Errorf("with Show Format on the override must be suppressed, got %+v", suppressed)
	}
	if len(suppressed.Pattern) != len(SketchEntityPattern(sk, l)) {
		t.Error("suppressed geometry must fall back to the sketch's default pattern")
	}
}

// An unstyled entity resolves the same either way — there is nothing to suppress.
func TestShowFormatIsANoOpForUnstyledGeometry(t *testing.T) {
	_, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	if SketchEntityStyle(sk, l, false).LineWeight != SketchEntityStyle(sk, l, true).LineWeight {
		t.Error("an unstyled entity must resolve identically with the toggle either way")
	}
}

func TestShowFormatTogglePersistsInOptions(t *testing.T) {
	s, _ := sketchSession(t)
	if s.ShowFormat() {
		t.Fatal("Show Format starts off, so user formatting is visible")
	}
	s.ToggleShowFormat()
	if !s.ShowFormat() {
		t.Error("the toggle must take effect")
	}
	s.ToggleShowFormat()
	if s.ShowFormat() {
		t.Error("pressing again must restore user formatting")
	}
}
