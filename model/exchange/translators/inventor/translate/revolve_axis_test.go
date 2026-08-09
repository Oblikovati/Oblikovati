// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	"oblikovati.org/model/exchange/translators/inventor/ipt"
)

func ln(ax, ay, bx, by float64) ipt.Line {
	return ipt.Line{A: ipt.Point2D{X: ax, Y: ay}, B: ipt.Point2D{X: bx, Y: by}}
}

// TestAxisAlignedLineClassifies: a vertical and a horizontal line are both valid centrelines; an
// oblique one is not.
func TestAxisAlignedLineClassifies(t *testing.T) {
	if !verticalLine(ln(1, 0, 1, 5)) || !axisAlignedLine(ln(1, 0, 1, 5)) {
		t.Error("a constant-X line is vertical and axis-aligned")
	}
	if !horizontalLine(ln(0, 2, 9, 2)) || !axisAlignedLine(ln(0, 2, 9, 2)) {
		t.Error("a constant-Y line is horizontal and axis-aligned")
	}
	if axisAlignedLine(ln(0, 0, 3, 5)) {
		t.Error("an oblique line is not axis-aligned")
	}
}

// TestProfileOneSideOfHorizontalAxis: a profile entirely above a horizontal centreline is one-sided;
// one straddling it is not — measured across Y for a horizontal axis.
func TestProfileOneSideOfHorizontalAxis(t *testing.T) {
	// axis at index 0 runs along y=0; profile lines sit at y>0.
	above := []ipt.Line{ln(0, 0, -1.8, 0), ln(-1.8, 0.9, 0, 0.9), ln(0, 0.9, 0, 1.3)}
	if !profileOneSideOfAxis(above, 0) {
		t.Error("a profile wholly above a y=0 axis should be one-sided")
	}
	straddle := []ipt.Line{ln(0, 0, -1.8, 0), ln(-1.8, -0.5, 0, 0.9)}
	if profileOneSideOfAxis(straddle, 0) {
		t.Error("a profile crossing the y=0 axis is not one-sided")
	}
}
