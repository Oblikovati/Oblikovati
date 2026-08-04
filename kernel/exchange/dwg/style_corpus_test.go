// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// The per-entity formatting must come out of REAL AutoCAD files as plausible values, not as
// garbage from a misaligned bit stream (#2015). The decoder always read the colour and line
// weight to stay aligned and discarded them; capturing them is only correct if what surfaces is
// in range.
//
// This is an oracle test against the git-ignored corpus: it skips cleanly when the files are
// absent, so the unit suite stays self-contained.
func TestCorpusEntityStylesAreInRange(t *testing.T) {
	for _, name := range append([]string{"testfile-2.dwg"}, r2018Corpus...) {
		t.Run(name, func(t *testing.T) {
			dr, _, err := Decode(loadTestFile(t, name))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			for handle, s := range dr.Styles {
				assertPlausibleStyle(t, handle, s)
			}
		})
	}
}

// assertPlausibleStyle checks one recorded style against the format's own ranges. A misaligned
// read shows up here as a colour of 40000 or a negative weight, which is what makes this a real
// check rather than a smoke test.
func assertPlausibleStyle(t *testing.T, handle uint64, s drawing.Style) {
	t.Helper()
	if s.Color < 0 || (s.Color > 255 && s.Color != drawing.ColorByLayer && s.Color != drawing.ColorByBlock) {
		t.Errorf("handle %d: colour index %d is outside the ACI range and is not a sentinel", handle, s.Color)
	}
	if s.LineWeight != drawing.LineWeightByLayer && (s.LineWeight < 0 || s.LineWeight > 211) {
		t.Errorf("handle %d: line weight %d is outside the standard 0..211 range", handle, s.LineWeight)
	}
}

// A drawing of entities that all inherit records no styles at all, so the common case costs
// nothing.
func TestInheritedFormattingRecordsNoStyle(t *testing.T) {
	c := &collector{}
	c.recordStyle(1, commonEntity{colorIndex: dwgColorByLayer, lineWeight: dwgLineWeightByLayer})
	if len(c.styles) != 0 {
		t.Errorf("styles = %d, want 0 — an entity that inherits everything is not recorded", len(c.styles))
	}
}

// An explicit colour is recorded.
func TestExplicitColourIsRecorded(t *testing.T) {
	c := &collector{}
	c.recordStyle(1, commonEntity{colorIndex: 5, lineWeight: dwgLineWeightByLayer})
	if got := c.styles[1].Color; got != 5 {
		t.Errorf("colour = %d, want 5", got)
	}
}

// The line-weight code table maps the sentinels to inherit and real codes to their widths.
func TestLineWeightCodes(t *testing.T) {
	if got := dwgLineWeightValue(0x1D); got != dwgLineWeightByLayer {
		t.Errorf("BYLAYER code = %d, want inherit", got)
	}
	if got := dwgLineWeightValue(3); got != 13 {
		t.Errorf("code 3 = %d, want 13 hundredths of a millimetre", got)
	}
}
