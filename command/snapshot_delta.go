// SPDX-License-Identifier: GPL-2.0-only

package command

import "fmt"

// snapshotDelta is a forward byte patch from one recipe snapshot to the next, encoded as the
// length of the common prefix, the length of the common suffix, and the differing middle that
// replaces everything between them. Consecutive undo snapshots differ only where the model
// actually changed — and the recipe codec is deterministic (snapshot.go), so a single edit
// produces a localised change in the serialized bytes. The middle is therefore O(edit size),
// not O(model size), which is what turns the per-edit whole-recipe storage into an O(N) stream
// (Oblikovati#1424). A non-localised edit (e.g. a parameter rename touching many references)
// stores a larger middle but is still correct — prefix/suffix is a lossless framing.
type snapshotDelta struct {
	prefix int    // bytes shared with the source snapshot at the start
	suffix int    // bytes shared with the source snapshot at the end (non-overlapping with prefix)
	middle []byte // the bytes that replace source[prefix : len(source)-suffix]
}

// makeSnapshotDelta computes the forward patch that turns from into to.
//
// Example:
//
//	d := makeSnapshotDelta([]byte(`{"d":10}`), []byte(`{"d":25}`)) // middle == "25"
//	got, _ := d.apply([]byte(`{"d":10}`))                          // got == {"d":25}
func makeSnapshotDelta(from, to []byte) snapshotDelta {
	p := commonPrefixLen(from, to)
	// Measure the suffix only within the bytes that follow the shared prefix, so prefix and
	// suffix can never overlap and double-count a shared byte (which would corrupt apply).
	s := commonSuffixLen(from[p:], to[p:])
	middle := append([]byte(nil), to[p:len(to)-s]...)
	return snapshotDelta{prefix: p, suffix: s, middle: middle}
}

// apply reconstructs the target snapshot by splicing the stored middle between from's shared
// prefix and suffix. It errors if from is not the snapshot this delta was built against (its
// length cannot accommodate the prefix+suffix), so a corrupt stream surfaces instead of
// silently producing a malformed recipe.
func (d snapshotDelta) apply(from []byte) ([]byte, error) {
	if d.prefix+d.suffix > len(from) {
		return nil, fmt.Errorf("command: snapshot delta prefix %d + suffix %d exceeds source length %d", d.prefix, d.suffix, len(from))
	}
	out := make([]byte, 0, d.prefix+len(d.middle)+d.suffix)
	out = append(out, from[:d.prefix]...)
	out = append(out, d.middle...)
	out = append(out, from[len(from)-d.suffix:]...)
	return out, nil
}

// size is the delta's retained byte cost — the differing middle plus the two length fields —
// used by the stream's memory accounting (and its O(N)-growth regression test).
func (d snapshotDelta) size() int { return len(d.middle) + 2*8 }

// commonPrefixLen returns the number of leading bytes a and b share.
func commonPrefixLen(a, b []byte) int {
	n := min(len(b), len(a))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// commonSuffixLen returns the number of trailing bytes a and b share.
func commonSuffixLen(a, b []byte) int {
	n := min(len(b), len(a))
	for i := range n {
		if a[len(a)-1-i] != b[len(b)-1-i] {
			return i
		}
	}
	return n
}
