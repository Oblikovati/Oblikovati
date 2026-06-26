// SPDX-License-Identifier: GPL-2.0-only

package command

import (
	"bytes"
	"testing"
)

// roundTrip is the codec's defining property: apply(makeDelta(from,to), from) == to, for any
// pair. Every case below asserts it, so a regression in either half is caught.
func roundTrip(t *testing.T, from, to []byte) {
	t.Helper()
	d := makeSnapshotDelta(from, to)
	got, err := d.apply(from)
	if err != nil {
		t.Fatalf("apply(makeDelta(%q,%q)): %v", from, to, err)
	}
	if !bytes.Equal(got, to) {
		t.Fatalf("apply(makeDelta(%q,%q)) = %q, want %q", from, to, got, to)
	}
}

func TestSnapshotDeltaRoundTrips(t *testing.T) {
	cases := []struct{ from, to string }{
		{"", ""},                           // empty → empty
		{"abc", "abc"},                     // identical
		{"", "abc"},                        // grow from nothing
		{"abc", ""},                        // shrink to nothing
		{`{"d":10}`, `{"d":25}`},           // localised middle edit (equal length)
		{`{"d":10}`, `{"d":250}`},          // localised middle edit (grows)
		{`[1,2,3]`, `[1,2,3,4]`},           // append (prefix is everything but the close)
		{`[1,2,3,4]`, `[1,2,3]`},           // delete tail element
		{`{"a":1,"b":2}`, `{"a":9,"b":2}`}, // change at the front
		{`{"a":1,"b":2}`, `{"a":1,"b":9}`}, // change at the back
		{"aXa", "aYa"},                     // prefix and suffix both one byte, no overlap
		{"aaaa", "aa"},                     // overlap-prone: shared bytes must not double-count
		{"aa", "aaaa"},                     // its inverse
	}
	for _, c := range cases {
		roundTrip(t, []byte(c.from), []byte(c.to))
	}
}

// TestSnapshotDeltaLocalisedEditIsSmall pins the whole point: a one-field change in a large
// snapshot stores a middle proportional to the edit, not to the snapshot.
func TestSnapshotDeltaLocalisedEditIsSmall(t *testing.T) {
	big := bytes.Repeat([]byte("feature,"), 1000) // ~8 KB
	from := append(append([]byte(`{"head":"`), big...), []byte(`,"d":10}`)...)
	to := append(append([]byte(`{"head":"`), big...), []byte(`,"d":25}`)...)
	d := makeSnapshotDelta(from, to)
	if len(d.middle) > 8 {
		t.Errorf("middle = %d bytes for a 2-byte edit in an %d-byte snapshot; delta is not localised", len(d.middle), len(from))
	}
	roundTrip(t, from, to)
}

// TestSnapshotDeltaApplyRejectsWrongSource: applying a delta to bytes it was not built against
// must error rather than fabricate a corrupt snapshot.
func TestSnapshotDeltaApplyRejectsWrongSource(t *testing.T) {
	d := makeSnapshotDelta([]byte("a-long-source-string"), []byte("a-long-target-string"))
	if _, err := d.apply([]byte("xy")); err == nil {
		t.Error("apply against a too-short source returned no error; a corrupt stream would go undetected")
	}
}
