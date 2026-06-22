// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/doc"
)

// zeroDocSession returns a session with the standard commands wired but no document open, so
// BuildRibbon selects the ZeroDoc ribbon.
func zeroDocSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	return s
}

func TestRibbonKeyForDocument(t *testing.T) {
	if got := ribbonKeyForDocument(nil); got != ZeroDocRibbon {
		t.Errorf("no document → %q, want ZeroDoc", got)
	}
	cases := map[doc.DocumentType]RibbonKey{
		doc.Part:         PartRibbon,
		doc.Assembly:     AssemblyRibbon,
		doc.Drawing:      DrawingRibbon,
		doc.Presentation: PresentationRibbon,
		doc.Unknown:      UnknownDocumentRibbon,
	}
	for dt, want := range cases {
		if got := ribbonKeyForDocument(doc.NewReference(dt, "d.obk")); got != want {
			t.Errorf("%v → %q, want %q", dt, got, want)
		}
	}
}

// TestZeroDocRibbonWhenNoDocumentOpen: with nothing open the ribbon is ZeroDoc — its Get
// Started tab offers New Part, and the part-only modeling tabs are absent (not greyed).
func TestZeroDocRibbonWhenNoDocumentOpen(t *testing.T) {
	r := BuildRibbon(zeroDocSession(t))
	if r.Key != ZeroDocRibbon {
		t.Fatalf("ribbon key = %q, want ZeroDoc", r.Key)
	}
	if _, ok := r.Tab("Get Started"); !ok {
		t.Error("ZeroDoc ribbon has no Get Started tab")
	}
	if _, ok := r.Panel("Launch"); !ok {
		t.Error("Get Started tab has no Launch panel with New Part")
	}
	if _, ok := r.Tab("Create & Modify"); ok {
		t.Error("the part-only 3D Model tab should be absent on the ZeroDoc ribbon")
	}
}

// TestPartRibbonWhenPartActive: opening a part switches the ribbon to the Part ribbon, which
// carries the modeling tabs but not the ZeroDoc Get Started tab.
func TestPartRibbonWhenPartActive(t *testing.T) {
	r := BuildRibbon(registeredSession(t))
	if r.Key != PartRibbon {
		t.Fatalf("ribbon key = %q, want Part", r.Key)
	}
	if _, ok := r.Tab("Create & Modify"); !ok {
		t.Error("Part ribbon has no 3D Model tab")
	}
	if _, ok := r.Tab("Get Started"); ok {
		t.Error("the ZeroDoc Get Started tab should not appear on the Part ribbon")
	}
}

// TestSketchTabIsContextual: the Sketch tab is absent in the part environment and appears only
// once a sketch is open (Inventor's contextual tab).
func TestSketchTabIsContextual(t *testing.T) {
	s := registeredSession(t)
	if _, ok := BuildRibbon(s).Tab("Sketch"); ok {
		t.Error("Sketch tab should be absent outside the sketch environment")
	}
	enterSketchEnv(t, s)
	if _, ok := BuildRibbon(s).Tab("Sketch"); !ok {
		t.Error("Sketch tab should appear inside the sketch environment")
	}
}

// TestNewPartCommandSwitchesRibbon: running New Part from the ZeroDoc ribbon creates a part
// and flips the active ribbon to Part — the document-driven ribbon switch.
func TestNewPartCommandSwitchesRibbon(t *testing.T) {
	s := zeroDocSession(t)
	if err := s.Execute("GetStarted.NewPart"); err != nil {
		t.Fatalf("New Part: %v", err)
	}
	if s.ActiveDocument() == nil || s.ActiveDocument().DocumentType() != doc.Part {
		t.Fatal("New Part did not create and activate a part document")
	}
	if r := BuildRibbon(s); r.Key != PartRibbon {
		t.Errorf("after New Part the ribbon is %q, want Part", r.Key)
	}
}

func TestEnvironmentShows(t *testing.T) {
	if !environmentShows(BaseEnvironment, SketchEnvironment) {
		t.Error("base-environment commands must show in any environment")
	}
	if environmentShows(SketchEnvironment, BaseEnvironment) {
		t.Error("a contextual command must not show outside its environment")
	}
	if !environmentShows(SketchEnvironment, SketchEnvironment) {
		t.Error("a contextual command must show in its own environment")
	}
}
