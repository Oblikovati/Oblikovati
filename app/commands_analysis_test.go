// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestPhysicalPropertiesNotice: the Physical Properties command computes the active part's mass
// properties and reports them in the status bar.
func TestPhysicalPropertiesNotice(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("commands: %v", err)
	}
	part, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := part.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(gmath.P2(0, 0), 2)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	if err := physicalProperties(s); err != nil {
		t.Fatalf("physicalProperties: %v", err)
	}
	notice := s.Notice()
	if !strings.Contains(notice, "Physical Properties") || !strings.Contains(notice, "volume") {
		t.Fatalf("notice = %q, want a Physical Properties summary", notice)
	}
}
