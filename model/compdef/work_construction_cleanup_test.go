// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "testing"

// TestRefTokenAppears pins the boundary rules of the recipe scan that detects a feature's retained
// datum reference: a ref matches only as a whole token, never as the prefix of a longer ref
// ("plane/5" must not match inside "plane/50") nor inside a larger identifier (#1849).
func TestRefTokenAppears(t *testing.T) {
	cases := []struct {
		hay  string
		ref  string
		want bool
	}{
		{"plane: plane/5\n", "plane/5", true}, // a feature's plane field
		{"toPlane: plane/5\nfrom: axis/1\n", "axis/1", true},
		{"plane: plane/50\n", "plane/5", false},         // trailing digit extends the ref
		{"plane: plane/5\n", "plane/50", false},         // the longer ref is absent
		{"note: myplane/5x\n", "plane/5", false},        // leading letter + trailing letter: not a token
		{"", "plane/3", false},                          // empty recipe (no features)
		{"a: point/12\nb: plane/3\n", "point/12", true}, // multi-line, exact token
		{"a: point/120\n", "point/12", false},           // trailing digit again
	}
	for _, c := range cases {
		if got := refTokenAppears([]byte(c.hay), c.ref); got != c.want {
			t.Errorf("refTokenAppears(%q, %q) = %v, want %v", c.hay, c.ref, got, c.want)
		}
	}
}
