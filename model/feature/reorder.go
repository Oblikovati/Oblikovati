// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Reorder moves a feature to a new history index, rejecting any move that would
// place it before a feature it depends on (or after a feature that depends on it).
// A valid reorder marks the affected range dirty so the next recompute replays it.
func (fs *PartFeatures) Reorder(f *PartFeature, newIndex int) error {
	old := fs.indexOf(f)
	if old < 0 {
		return fmt.Errorf("feature: %q is not in this program", f.name)
	}
	if newIndex < 0 || newIndex >= len(fs.items) {
		return fmt.Errorf("feature: reorder index %d out of range [0,%d)", newIndex, len(fs.items))
	}
	proposed := moved(fs.items, old, newIndex)
	if bad := firstDependencyViolation(proposed); bad != nil {
		return fmt.Errorf("feature: reorder would place %q before its dependency", bad.name)
	}
	fs.items = proposed
	fs.markDirtyFrom(lowerOf(old, newIndex))
	return nil
}

// indexOf returns f's position, or -1.
func (fs *PartFeatures) indexOf(f *PartFeature) int {
	for i, item := range fs.items {
		if item == f {
			return i
		}
	}
	return -1
}

// moved returns a copy of items with the element at from relocated to to.
func moved(items []*PartFeature, from, to int) []*PartFeature {
	out := make([]*PartFeature, 0, len(items))
	out = append(out, items[:from]...)
	out = append(out, items[from+1:]...)
	insert := append([]*PartFeature{}, out[:to]...)
	insert = append(insert, items[from])
	return append(insert, out[to:]...)
}

// firstDependencyViolation returns the first feature whose dependency appears after
// it in the order, or nil if the order is dependency-valid.
func firstDependencyViolation(order []*PartFeature) *PartFeature {
	pos := make(map[ID]int, len(order))
	for i, f := range order {
		pos[f.id] = i
	}
	for i, f := range order {
		for _, d := range f.deps {
			if j, ok := pos[d]; ok && j >= i {
				return f
			}
		}
	}
	return nil
}

// markDirtyFrom marks every feature at or after index i dirty.
func (fs *PartFeatures) markDirtyFrom(i int) {
	for ; i < len(fs.items); i++ {
		fs.items[i].dirty = true
	}
}

// SetEndOfPart moves the end-of-part marker before the given feature, so features
// from it onward are excluded from evaluation (rolled back). A nil feature, or
// RollToEnd, restores full evaluation.
func (fs *PartFeatures) SetEndOfPart(f *PartFeature) error {
	i := fs.indexOf(f)
	if i < 0 {
		return fmt.Errorf("feature: %q is not in this program", f.name)
	}
	fs.eop = i
	fs.forceReplay()
	return nil
}

// RollToEnd restores evaluation of the whole program.
func (fs *PartFeatures) RollToEnd() {
	fs.eop = eopAll
	fs.forceReplay()
}

// IsRolledBack reports whether the EOP marker excludes any trailing features.
func (fs *PartFeatures) IsRolledBack() bool { return fs.eop != eopAll && fs.eop < len(fs.items) }

// EndOfPartIndex returns the marker position, or -1 when at the end.
func (fs *PartFeatures) EndOfPartIndex() int { return fs.eop }

// forceReplay marks the program for a full replay (the result depends on where the
// marker sits, so a clean program must still be re-evaluated to the new cutoff).
func (fs *PartFeatures) forceReplay() {
	if len(fs.items) > 0 {
		fs.items[0].dirty = true
	}
}

func lowerOf(a, b int) int {
	if a < b {
		return a
	}
	return b
}
