// SPDX-License-Identifier: GPL-2.0-only

package dwg

// The handle stream is a per-object region (starting at entityCursor.handleStart) that
// holds the object's handle references — owner, layer, colour, and, for an INSERT, the
// block it places. References are encoded relative to the object's own handle, so they
// are resolved against it (ODA dwg_decode_handleref). Reading them needs the
// common-entity flags captured in entitydata.go, since each flag gates a handle.

// resolveHandle turns a (code, value) handle reference into an absolute handle, using
// the referencing object's own handle for the relative codes (6/8/A/C/E); codes 2–5
// and 0 are already absolute.
func resolveHandle(code uint8, value, own uint64) uint64 {
	switch code {
	case 0x06:
		return own + 1
	case 0x08:
		return own - 1
	case 0x0A:
		return own + value
	case 0x0C:
		return own - value
	case 0x0E:
		return own
	default: // 0, 2, 3, 4, 5
		return value
	}
}

// readResolvedHandle reads one handle reference and resolves it against own.
func readResolvedHandle(r *BitReader, own uint64) uint64 {
	h := r.ReadHandle()
	return resolveHandle(h.Code, h.Value, own)
}

// commonEntityHandles walks the common-entity handle references in spec order and
// returns the owner handle (the owning block for entmode 0; 0 otherwise) plus a reader
// positioned right after them, so an entity-specific handle (e.g. INSERT.block_header)
// can be read next. The reader is independent of the geometry/data-stream cursor.
//
//nolint:funlen,gocyclo // sequential flag-gated handle reads in spec order; length/branches are the format.
func commonEntityHandles(r *BitReader, data []byte, cur *entityCursor, version Version) (owner uint64, _ *BitReader) {
	r.Reset(data, cur.handleStart)
	ce := cur.common
	own := cur.ownHandle
	if ce.colorByHandle {
		readResolvedHandle(r, own) // colour (DBCOLOR) handle comes first (read during ENC)
	}
	if ce.entmode == 0 {
		owner = readResolvedHandle(r, own)
	}
	for i := 0; i < ce.numReactors; i++ {
		readResolvedHandle(r, own)
	}
	if version >= R2004 {
		if !ce.xdicMissing {
			readResolvedHandle(r, own) // xdicobjhandle
		}
	} else {
		readResolvedHandle(r, own) // pre-R2004 always has xdicobjhandle
	}
	if version <= R2000 && !ce.noLinks {
		readResolvedHandle(r, own) // prev_entity
		readResolvedHandle(r, own) // next_entity
	}
	readResolvedHandle(r, own) // layer (always present R2000+)
	if ce.ltypeFlags == 3 {
		readResolvedHandle(r, own) // ltype
	}
	if version >= R2007 {
		if ce.materialFlags == 3 {
			readResolvedHandle(r, own) // material
		}
		if ce.shadowHandle {
			readResolvedHandle(r, own) // shadow
		}
	}
	if ce.plotstyleFlags == 3 {
		readResolvedHandle(r, own) // plotstyle
	}
	if version >= R2010 {
		if ce.hasFullVS {
			readResolvedHandle(r, own)
		}
		if ce.hasFaceVS {
			readResolvedHandle(r, own)
		}
		if ce.hasEdgeVS {
			readResolvedHandle(r, own)
		}
	}
	return owner, r
}
