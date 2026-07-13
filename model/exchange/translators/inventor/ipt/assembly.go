// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "strconv"

// Assembly (.iam) decoding. An assembly's segments carry the "Am" (assembly) prefix rather
// than "Pm" (part). Its component STRUCTURE lives in AmDcSegment, where each placed
// component is a "component:instance" occurrence name (e.g. "asm_box:2"). Each occurrence's
// PLACEMENT transform lives in AmRxSegment (the resolved-references segment) as a typed
// node (232792BC) carrying a sparse Transformation3D — reached through the node-graph
// walk (nodegraph.go). DecodeAssembly zips the two into placed occurrences.

// Occurrence is a decoded assembly occurrence: the base name of the component it instances
// (e.g. "asm_box" → asm_box.ipt) and its 1-based instance number.
type Occurrence struct {
	Component string
	Instance  int
}

// PlacedOccurrence pairs an occurrence with its placement transform (model space, cm) —
// the assembly's component structure zipped with its resolved placement.
type PlacedOccurrence struct {
	Component string
	Instance  int
	Transform Matrix4
}

// occurrenceNodeType is the AmRxSegment node holding one occurrence's resolved placement
// transform (InventorLoader Read_232792BC); occurrenceXfOffset is where the sparse
// Transformation3D begins in that node's payload (Header0 = 6 bytes + a child ref = 4).
const (
	occurrenceNodeType = 0x232792BC
	occurrenceXfOffset = 10
)

// DecodeAssembly returns the assembly's placed occurrences with no component filtering —
// suitable for assemblies without constraints. For assemblies that carry constraints (whose
// geometry selections emit spurious "hash:N" names into the same segment), use
// PlacedOccurrences with a keep predicate that rejects non-component names.
func DecodeAssembly(d *Document) []PlacedOccurrence {
	return d.PlacedOccurrences(nil)
}

// PlacedOccurrences pairs each occurrence with its placement transform. Occurrence names
// (component:instance) are scanned from AmDcSegment; keep (when non-nil) filters them by
// component so spurious names from constraint geometry selections don't corrupt the
// name↔transform alignment. The kept occurrences are matched to the authoritative
// AmRxSegment placement transforms in creation order.
func (d *Document) PlacedOccurrences(keep func(component string) bool) []PlacedOccurrence {
	seg, _ := d.Segment("AmDcSegment")
	var kept []Occurrence
	for _, o := range DecodeOccurrences(seg) {
		if keep == nil || keep(o.Component) {
			kept = append(kept, o)
		}
	}
	xf := DecodeOccurrenceTransforms(d)
	out := make([]PlacedOccurrence, len(kept))
	for i, o := range kept {
		t := identityMatrix4()
		if i < len(xf) {
			t = xf[i]
		}
		out[i] = PlacedOccurrence{Component: o.Component, Instance: o.Instance, Transform: t}
	}
	return out
}

// DecodeOccurrenceTransforms returns each occurrence's placement transform in creation
// order, read from the AmRxSegment placement nodes. Empty if the segment or its metadata is
// missing/unreadable.
func DecodeOccurrenceTransforms(d *Document) []Matrix4 {
	var out []Matrix4
	d.walkSegment("AmRxSegment", func(typ uint32, pay []byte) bool {
		if typ == occurrenceNodeType {
			if m, ok := decodeTransform3D(pay, occurrenceXfOffset); ok {
				out = append(out, m)
			}
		}
		return true
	})
	return out
}

// IsAssembly reports whether the document is an assembly (.iam) — it carries the assembly
// content segment AmDcSegment instead of a part's PmDCSegment.
func (d *Document) IsAssembly() bool {
	_, ok := d.Segment("AmDcSegment")
	return ok
}

// DecodeOccurrences lists the component occurrences named in AmDcSegment. Each occurrence
// is a UTF-16 "component:instance" name (e.g. "asm_box:2"); we scan for the ":<digits>"
// marker and read the component identifier backwards, so a name embedded in a longer run
// (adjacent bytes) still resolves. Deduped by name, in file (creation) order so it aligns
// with the AmRxSegment placement transforms. Component structure only; use DecodeAssembly
// to pair each occurrence with its transform.
func DecodeOccurrences(seg []byte) []Occurrence {
	var occ []Occurrence
	seen := map[string]bool{}
	// Names may start at an odd byte offset, so probe every offset (not just even) for the
	// UTF-16 ':' marker; the digit/identifier scans then step by 2 relative to it.
	for i := 0; i+2 <= len(seg); i++ {
		if seg[i] != ':' || seg[i+1] != 0 {
			continue
		}
		inst, end := utf16Uint(seg, i+2)
		if end == i+2 {
			continue // no digits after the colon
		}
		comp := utf16IdentBefore(seg, i)
		if comp == "" {
			continue
		}
		if key := comp + string(seg[i:end]); !seen[key] {
			seen[key] = true
			occ = append(occ, Occurrence{Component: comp, Instance: inst})
		}
	}
	return occ
}

// utf16Uint reads consecutive UTF-16LE ASCII digits from off, returning their value and the
// byte offset just past them (== off when there are no digits).
func utf16Uint(seg []byte, off int) (int, int) {
	var digits []byte
	i := off
	for i+2 <= len(seg) && seg[i+1] == 0 && seg[i] >= '0' && seg[i] <= '9' {
		digits = append(digits, seg[i])
		i += 2
	}
	if len(digits) == 0 {
		return 0, off
	}
	n, _ := strconv.Atoi(string(digits))
	return n, i
}

// utf16IdentBefore reads a UTF-16LE identifier ([A-Za-z0-9_-]) ending just before off,
// walking backwards. Returns "" when no identifier char precedes off.
func utf16IdentBefore(seg []byte, off int) string {
	var rev []byte
	for k := off - 2; k >= 0 && seg[k+1] == 0 && isIdentByte(seg[k]); k -= 2 {
		rev = append(rev, seg[k])
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return string(rev)
}

// isIdentByte reports whether b is a component-name identifier character.
func isIdentByte(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '_' || b == '-'
}

// ComponentRefs returns the distinct component base names referenced by the occurrences,
// in first-seen order — one per source file to translate (asm_box → asm_box.ipt).
func ComponentRefs(occ []Occurrence) []string {
	seen := map[string]bool{}
	var out []string
	for _, o := range occ {
		if !seen[o.Component] {
			seen[o.Component] = true
			out = append(out, o.Component)
		}
	}
	return out
}
