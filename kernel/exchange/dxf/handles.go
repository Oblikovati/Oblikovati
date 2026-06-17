// SPDX-License-Identifier: GPL-2.0-only

package dxf

// encHandles holds the handles assigned to every fixed object the encoder writes (tables,
// table records, block records, blocks, the root dictionary) plus the base handle for the
// model-space entities. DXF handles are arbitrary unique hex ids; the relationships between
// them (a record's 330 owner, an entity's 330 owner) are what matter, so they are allocated
// once up front and referenced by name. Handle 0 is reserved to mean "no owner / the
// drawing", matching how AutoCAD writes table owners.
type encHandles struct {
	next uint64

	// Symbol tables.
	blockRecordTable, vportTable, ltypeTable, layerTable, styleTable uint64
	viewTable, ucsTable, appidTable, dimstyleTable                   uint64

	// Standard table records.
	vportActive                                   uint64
	ltypeByBlock, ltypeByLayer, ltypeContinuous   uint64
	layer0, styleStandard, appidACAD, dimstyleStd uint64
	modelSpaceBR, paperSpaceBR                    uint64

	// Block definitions (BLOCK/ENDBLK markers).
	modelBlock, modelEndblk, paperBlock, paperEndblk uint64

	// Named-object dictionary.
	rootDict uint64

	// First handle for the model-space entities; each entity takes the next in sequence.
	entityBase uint64
}

// alloc returns the next free handle.
func (h *encHandles) alloc() uint64 {
	h.next++
	return h.next
}

// newEncHandles allocates handles for every fixed object in a deterministic order (so the
// output is stable) and sets entityBase to the first handle the entities will use.
//
//nolint:funlen // one allocation per fixed object; the list is the object graph.
func newEncHandles() *encHandles {
	h := &encHandles{}
	h.blockRecordTable = h.alloc()
	h.vportTable = h.alloc()
	h.ltypeTable = h.alloc()
	h.layerTable = h.alloc()
	h.styleTable = h.alloc()
	h.viewTable = h.alloc()
	h.ucsTable = h.alloc()
	h.appidTable = h.alloc()
	h.dimstyleTable = h.alloc()

	h.vportActive = h.alloc()
	h.ltypeByBlock = h.alloc()
	h.ltypeByLayer = h.alloc()
	h.ltypeContinuous = h.alloc()
	h.layer0 = h.alloc()
	h.styleStandard = h.alloc()
	h.appidACAD = h.alloc()
	h.dimstyleStd = h.alloc()

	h.modelSpaceBR = h.alloc()
	h.paperSpaceBR = h.alloc()
	h.modelBlock = h.alloc()
	h.modelEndblk = h.alloc()
	h.paperBlock = h.alloc()
	h.paperEndblk = h.alloc()

	h.rootDict = h.alloc()

	h.entityBase = h.next + 1
	return h
}
