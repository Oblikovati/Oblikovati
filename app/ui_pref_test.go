// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestUIScaleDefaultsToFullSize(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if got := s.UIFontScale(); got != 1 {
		t.Errorf("UIFontScale default = %v, want 1", got)
	}
	if got := s.UIIconScale(); got != 1 {
		t.Errorf("UIIconScale default = %v, want 1", got)
	}
}

func TestSetUIScaleStoresAndReadsBack(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.SetUIFontScale(1.25); err != nil {
		t.Fatalf("SetUIFontScale: %v", err)
	}
	if err := s.SetUIIconScale(1.75); err != nil {
		t.Fatalf("SetUIIconScale: %v", err)
	}
	if got := s.UIFontScale(); got != 1.25 {
		t.Errorf("UIFontScale = %v, want 1.25", got)
	}
	if got := s.UIIconScale(); got != 1.75 {
		t.Errorf("UIIconScale = %v, want 1.75", got)
	}
}

func TestSetUIScaleClampsToBounds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		set       func(*Session, float64) error
		read      func(*Session) float64
		in, wantC float64
	}{
		{"font below min", (*Session).SetUIFontScale, (*Session).UIFontScale, 0.1, minUIScale},
		{"font above max", (*Session).SetUIFontScale, (*Session).UIFontScale, 9, maxUIFontScale},
		{"icon below min", (*Session).SetUIIconScale, (*Session).UIIconScale, -2, minUIScale},
		{"icon above max", (*Session).SetUIIconScale, (*Session).UIIconScale, 9, maxUIIconScale},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewSession()
			if err := c.set(s, c.in); err != nil {
				t.Fatalf("set(%v): %v", c.in, err)
			}
			if got := c.read(s); got != c.wantC {
				t.Errorf("after set(%v) = %v, want clamped %v", c.in, got, c.wantC)
			}
		})
	}
}

// A non-positive stored scale (corrupt/edited options file) must read back as full size so the
// interface can never be hidden — distinct from clampScale, which the explicit setters use.
func TestUIScaleReadGuardsCorruptZero(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.appOptions.UI.FontScale = 0
	s.appOptions.UI.IconScale = -1
	if got := s.UIFontScale(); got != 1 {
		t.Errorf("UIFontScale(zero) = %v, want 1", got)
	}
	if got := s.UIIconScale(); got != 1 {
		t.Errorf("UIIconScale(negative) = %v, want 1", got)
	}
}
