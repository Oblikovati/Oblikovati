// SPDX-License-Identifier: GPL-2.0-only

package cmdline

import (
	"math"
	"testing"
)

func TestFieldsSplitsCommandAndCoords(t *testing.T) {
	got := Fields("  LINE 0,0   10,0 ")
	want := []string{"LINE", "0,0", "10,0"}
	if len(got) != len(want) {
		t.Fatalf("Fields = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Fields[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseCoordCartesian(t *testing.T) {
	c, ok := ParseCoord("10,5")
	if !ok || c.X != 10 || c.Y != 5 || c.Z != 0 || c.Relative {
		t.Errorf("ParseCoord(10,5) = %+v,%v", c, ok)
	}
	c3, ok := ParseCoord("1,2,3")
	if !ok || c3.X != 1 || c3.Y != 2 || c3.Z != 3 {
		t.Errorf("ParseCoord(1,2,3) = %+v,%v", c3, ok)
	}
}

func TestParseCoordRelative(t *testing.T) {
	c, ok := ParseCoord("@10,0")
	if !ok || !c.Relative || c.X != 10 || c.Y != 0 {
		t.Errorf("ParseCoord(@10,0) = %+v,%v", c, ok)
	}
}

func TestParseCoordPolar(t *testing.T) {
	c, ok := ParseCoord("@10<90")
	if !ok || !c.Relative {
		t.Fatalf("ParseCoord(@10<90) = %+v,%v", c, ok)
	}
	if math.Abs(c.X) > 1e-9 || math.Abs(c.Y-10) > 1e-9 {
		t.Errorf("polar 10<90 = (%.4f,%.4f), want (0,10)", c.X, c.Y)
	}
	abs, ok := ParseCoord("5<0")
	if !ok || abs.Relative || math.Abs(abs.X-5) > 1e-9 || math.Abs(abs.Y) > 1e-9 {
		t.Errorf("polar 5<0 = %+v,%v, want X=5 Y=0 absolute", abs, ok)
	}
}

func TestParseCoordRejectsNonCoord(t *testing.T) {
	for _, tok := range []string{"", "@", "Close", "1,", "a,b", "1,2,3,4"} {
		if _, ok := ParseCoord(tok); ok {
			t.Errorf("ParseCoord(%q) should fail", tok)
		}
	}
}

func TestParseDistance(t *testing.T) {
	if v, ok := ParseDistance("25.5"); !ok || v != 25.5 {
		t.Errorf("ParseDistance(25.5) = %v,%v", v, ok)
	}
	if _, ok := ParseDistance("close"); ok {
		t.Error("ParseDistance(close) should fail")
	}
}

func TestMatchKeyword(t *testing.T) {
	opts := []string{"Close", "Undo"}
	if k, ok := MatchKeyword("c", opts); !ok || k != "Close" {
		t.Errorf("MatchKeyword(c) = %q,%v, want Close", k, ok)
	}
	if k, ok := MatchKeyword("UNDO", opts); !ok || k != "Undo" {
		t.Errorf("MatchKeyword(UNDO) = %q,%v, want Undo", k, ok)
	}
	if _, ok := MatchKeyword("x", opts); ok {
		t.Error("MatchKeyword(x) should fail (no match)")
	}
}

func TestMatchKeywordAmbiguousPrefixFails(t *testing.T) {
	if _, ok := MatchKeyword("C", []string{"Close", "Corner"}); ok {
		t.Error("ambiguous prefix C should fail")
	}
}
