// SPDX-License-Identifier: GPL-2.0-only

package occurrence

// OccurrencePath addresses a (possibly nested) occurrence from a root assembly: the
// sequence of occurrence instance names from the top down, e.g.
// ["gearbox:1", "shaft:2"]. Because sub-occurrences are shared flyweights, the same
// nested occurrence object is reached through every placement of its sub-assembly; the
// path is what disambiguates the *context* it is reached through — the reference API's
// ComponentOccurrence.OccurrencePath. The empty path resolves to nothing.
type OccurrencePath []string

// Resolve walks the path from this collection down through nested sub-assemblies,
// returning the addressed occurrence. It fails (false) on an empty path, a segment
// naming no occurrence, or a non-final segment whose occurrence is a leaf part (no
// sub-occurrences to descend into).
func (c *Occurrences) Resolve(path OccurrencePath) (*Occurrence, bool) {
	chain, ok := c.ResolveChain(path)
	if !ok {
		return nil, false
	}
	return chain[len(chain)-1], true
}

// ResolveChain walks the path and returns every occurrence along it — the root
// segment first, the addressed occurrence last — so the caller has each occurrence's
// parent context (chain[len-2] is the addressed occurrence's parent occurrence). It
// fails identically to [Occurrences.Resolve].
func (c *Occurrences) ResolveChain(path OccurrencePath) ([]*Occurrence, bool) {
	if len(path) == 0 {
		return nil, false
	}
	chain := make([]*Occurrence, 0, len(path))
	level := c
	for i, name := range path {
		if level == nil {
			return nil, false // a leaf part has no sub-occurrences to descend into
		}
		o := level.byName(name)
		if o == nil {
			return nil, false
		}
		chain = append(chain, o)
		if i < len(path)-1 {
			level = o.SubOccurrences()
		}
	}
	return chain, true
}

// ParentInPath returns the occurrence that contains the path's target — its parent in
// this navigation context — or nil when the target is top-level (a single-segment
// path). Because a shared sub-occurrence has no single parent, the parent is a
// property of the path, not of the occurrence.
func (c *Occurrences) ParentInPath(path OccurrencePath) (*Occurrence, bool) {
	chain, ok := c.ResolveChain(path)
	if !ok {
		return nil, false
	}
	if len(chain) < 2 {
		return nil, true // resolved, but the target is top-level: no parent
	}
	return chain[len(chain)-2], true
}
