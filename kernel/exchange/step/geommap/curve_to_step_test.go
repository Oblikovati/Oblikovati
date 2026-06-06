// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"testing"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

func TestLineSegmentExportsAsLine(t *testing.T) {
	w := part21.NewWriter()
	e := NewEmitter(w, 1.0)
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	_, sameSense, err := e.CurveToStep(seg)
	if err != nil {
		t.Fatalf("CurveToStep segment: %v", err)
	}
	if !sameSense {
		t.Error("a line segment exports with same_sense=true")
	}
}

func TestCircleExportsSameSenseTrue(t *testing.T) {
	w := part21.NewWriter()
	e := NewEmitter(w, 1.0)
	c, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	_, sameSense, err := e.CurveToStep(c)
	if err != nil || !sameSense {
		t.Errorf("circle export sameSense=%v err=%v, want true,nil", sameSense, err)
	}
}

func TestClockwiseArcExportsSameSenseFalse(t *testing.T) {
	w := part21.NewWriter()
	e := NewEmitter(w, 1.0)
	// A negative sweep is clockwise about the normal → the STEP CIRCLE (always CCW)
	// must be referenced with same_sense=false.
	arc, _ := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, -1.0)
	_, sameSense, err := e.CurveToStep(arc)
	if err != nil {
		t.Fatalf("CurveToStep arc: %v", err)
	}
	if sameSense {
		t.Error("a clockwise arc must export same_sense=false")
	}
}
