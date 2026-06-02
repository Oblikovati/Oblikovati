// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

func TestDocumentTypeIsValid(t *testing.T) {
	for _, valid := range []DocumentType{Part, Assembly, Drawing, Presentation} {
		if !valid.IsValid() {
			t.Errorf("%v.IsValid() = false, want true", valid)
		}
	}
	for _, invalid := range []DocumentType{Unknown, DocumentType(99)} {
		if invalid.IsValid() {
			t.Errorf("%v.IsValid() = true, want false", invalid)
		}
	}
}

func TestDocumentTypeStringIsStable(t *testing.T) {
	cases := map[DocumentType]string{
		Unknown:          "unknown",
		Part:             "part",
		Assembly:         "assembly",
		Drawing:          "drawing",
		Presentation:     "presentation",
		DocumentType(99): "unknown",
	}
	for dt, want := range cases {
		if got := dt.String(); got != want {
			t.Errorf("DocumentType(%d).String() = %q, want %q", dt, got, want)
		}
	}
}

// The stable wire values are part of the persisted contract; guard against an
// accidental renumber (architecture core/05).
func TestDocumentTypeValuesAreStable(t *testing.T) {
	cases := map[DocumentType]uint32{Unknown: 0, Part: 1, Assembly: 2, Drawing: 3, Presentation: 4}
	for dt, want := range cases {
		if uint32(dt) != want {
			t.Errorf("DocumentType %v = %d, want stable %d", dt, uint32(dt), want)
		}
	}
}
