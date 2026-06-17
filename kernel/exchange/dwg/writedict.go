// SPDX-License-Identifier: GPL-2.0-only

package dwg

// writedict.go encodes DICTIONARY objects — the named-object dictionary (NOD) at the root of
// the object graph and any sub-dictionaries. A dictionary is a list of name→handle pairs
// (ODA §20.4.44): the names live in the data stream, the target handles (soft owner) in the
// handle stream, after the owner and xdic.

// dictEntry is one name→object mapping in a dictionary.
type dictEntry struct {
	name   string
	handle uint64
}

// TypeDictionary is the fixed R2000 type code for a DICTIONARY object (ODA §20.3).
const TypeDictionary ObjectType = 0x2A

// writeDictionary frames a DICTIONARY: cloning flag 1 (keep existing on name clash), not a
// hard-owner dictionary, the entry names, then the owner/xdic/item handles. owner is the
// dictionary's parent (0 = root, for the NOD).
func writeDictionary(handle, owner uint64, entries []dictEntry) []byte {
	b := newObjectBody(handle, int(TypeDictionary))
	b.data.WriteBL(0)            // numreactors
	b.data.WriteBL(len(entries)) // numitems
	b.data.WriteBS(1)            // cloning flag (keep existing)
	b.data.WriteRC(0)            // hard-owner flag (soft-owned items)
	for _, e := range entries {
		writeName(b.data, e.name)
	}
	b.handles.WriteHandle(softPtrCode, owner)
	b.handles.WriteHandle(hardOwnerCode, 0) // xdicobjhandle (null)
	for _, e := range entries {
		b.handles.WriteHandle(softOwnerCode, e.handle)
	}
	return frameObject(b)
}
