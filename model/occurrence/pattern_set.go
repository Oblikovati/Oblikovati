// SPDX-License-Identifier: GPL-2.0-only

package occurrence

// OccurrencePatternSet is the collection of PERSISTENT occurrence patterns an assembly owns
// (#1976). Before this, a pattern was built, its copies placed, and the pattern object thrown
// away — so a patterned array could not be re-read, suppressed, or deleted as a group. The set
// keeps each pattern, mints its id, and deletes an array (dropping the generated occurrences but
// keeping the seed) as one operation. It mirrors [Occurrences]' id-minting shape.
type OccurrencePatternSet struct {
	occs   *Occurrences
	items  []*OccurrencePattern
	nextID uint64
}

// NewOccurrencePatternSet binds the set to the assembly's occurrences, which it removes from on
// delete.
func NewOccurrencePatternSet(occs *Occurrences) *OccurrencePatternSet {
	return &OccurrencePatternSet{occs: occs}
}

// Add records a created pattern under name, assigns it the next id, binds its element
// occurrences (seed first, then the generated copies in element order), and returns it.
func (s *OccurrencePatternSet) Add(pat *OccurrencePattern, name string, seed *Occurrence, generated []*Occurrence) *OccurrencePattern {
	s.nextID++
	pat.id = s.nextID
	pat.name = name
	pat.BindOccurrences(seed, generated)
	s.items = append(s.items, pat)
	return pat
}

// Count returns how many patterns the assembly holds; Item returns the i-th in creation order.
func (s *OccurrencePatternSet) Count() int                    { return len(s.items) }
func (s *OccurrencePatternSet) Item(i int) *OccurrencePattern { return s.items[i] }

// ByID returns the pattern with the given id.
func (s *OccurrencePatternSet) ByID(id uint64) (*OccurrencePattern, bool) {
	for _, p := range s.items {
		if p.id == id {
			return p, true
		}
	}
	return nil, false
}

// Delete removes the pattern (by id) and the occurrences it generated — every element beyond the
// seed — leaving the seed component in place. It reports whether the pattern was present.
func (s *OccurrencePatternSet) Delete(id uint64) bool {
	for i, p := range s.items {
		if p.id != id {
			continue
		}
		for e := 1; e < len(p.occs); e++ { // element 0 is the seed; keep it
			if p.occs[e] != nil {
				s.occs.Remove(p.occs[e])
			}
		}
		s.items = append(s.items[:i], s.items[i+1:]...)
		return true
	}
	return false
}
