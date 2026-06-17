// SPDX-License-Identifier: GPL-2.0-only

package dwg

// writegraph.go synthesises the full R2000 system-object graph that AutoCAD requires in
// every drawing: the nine symbol-table control objects, their standard records (layer 0,
// the BYLAYER/BYBLOCK/CONTINUOUS linetypes, the STANDARD text/dim styles, the ACAD appid,
// the *Active viewport), the *Model_Space/*Paper_Space block records with their BLOCK/ENDBLK
// markers, and the named-object dictionary. The model-space entities are linked into the
// model-space block record. Handles are allocated up front (graphHandles) so every
// cross-reference resolves. Layouts/plotstyles are emitted by writedict.go.

// graphHandles holds the handle assigned to every system object, allocated in ascending
// order so the object map (which encodes ascending handle deltas) stays monotonic. Entities
// take the handles after blockEntityBase.
type graphHandles struct {
	blockControl, layerControl, styleControl, ltypeControl uint64
	viewControl, ucsControl, vportControl, appidControl    uint64
	dimstyleControl                                        uint64
	nod                                                    uint64
	layer0, styleStandard                                  uint64
	ltypeByBlock, ltypeByLayer, ltypeContinuous            uint64
	appidAcad, vportActive, dimstyleStandard               uint64
	modelHdr, paperHdr                                     uint64
	modelBlock, modelEndblk, paperBlock, paperEndblk       uint64
	groupDict, mlineDict, mlineStandard                    uint64
	layoutDict, plotSettingsDict, plotStyleDict            uint64
	placeholder, layoutModel, layoutPaper                  uint64
	entityBase                                             uint64 // first model-space entity handle
}

// allocate assigns every system-object handle in ascending order and returns the next free
// handle (the first entity handle). The order here is also the file write order.
//
//nolint:funlen // one handle assignment per system object; length is the object count.
func (h *graphHandles) allocate() {
	n := uint64(0)
	next := func() uint64 { n++; return n }
	h.blockControl = next()
	h.layerControl = next()
	h.styleControl = next()
	h.ltypeControl = next()
	h.viewControl = next()
	h.ucsControl = next()
	h.vportControl = next()
	h.appidControl = next()
	h.dimstyleControl = next()
	h.nod = next()
	h.layer0 = next()
	h.styleStandard = next()
	h.ltypeByBlock = next()
	h.ltypeByLayer = next()
	h.ltypeContinuous = next()
	h.appidAcad = next()
	h.vportActive = next()
	h.dimstyleStandard = next()
	h.modelHdr = next()
	h.paperHdr = next()
	h.modelBlock = next()
	h.modelEndblk = next()
	h.paperBlock = next()
	h.paperEndblk = next()
	h.allocateDictionaries(next)
	h.entityBase = next()
}

// allocateDictionaries assigns the named-object-dictionary chain handles (group, mlinestyle,
// plotstyle, layouts). Kept separate so the symbol-table allocation above stays readable.
func (h *graphHandles) allocateDictionaries(next func() uint64) {
	h.groupDict = next()
	h.mlineDict = next()
	h.mlineStandard = next()
	h.plotStyleDict = next()
	h.placeholder = next()
	h.layoutDict = next()
	h.layoutModel = next()
	h.layoutPaper = next()
	h.plotSettingsDict = next()
}

// Fixed R2000 type codes for the symbol-table control objects (ODA §20.3).
const (
	typeBlockControl    = 0x30
	typeLayerControl    = 0x32
	typeStyleControl    = 0x34
	typeLtypeControl    = 0x38
	typeViewControl     = 0x3C
	typeUcsControl      = 0x3E
	typeVportControl    = 0x40
	typeAppidControl    = 0x42
	typeDimstyleControl = 0x44
)

// encodeGraph encodes every object in ascending handle order — the nine control objects,
// the dictionary, the standard records, the block records and markers, and the model-space
// entities — returning the concatenated bytes and the handle→offset references (offsets
// relative to the first object). The caller rebases the offsets to the file position.
//
//nolint:funlen // a flat list of the system objects in fixed handle order; length is the graph.
func encodeGraph(h *graphHandles, entities []Entity) ([]byte, []ObjectRef, error) {
	var buf []byte
	var refs []ObjectRef
	add := func(handle uint64, b []byte) {
		refs = append(refs, ObjectRef{Handle: handle, Offset: int64(len(buf))})
		buf = append(buf, b...)
	}
	ents := writableEntities(entities)

	add(h.blockControl, writeBlockControl(h))
	add(h.layerControl, writeControlObject(h.layerControl, typeLayerControl, []uint64{h.layer0}))
	add(h.styleControl, writeControlObject(h.styleControl, typeStyleControl, []uint64{h.styleStandard}))
	add(h.ltypeControl, writeLtypeControl(h))
	add(h.viewControl, writeControlObject(h.viewControl, typeViewControl, nil))
	add(h.ucsControl, writeControlObject(h.ucsControl, typeUcsControl, nil))
	add(h.vportControl, writeControlObject(h.vportControl, typeVportControl, []uint64{h.vportActive}))
	add(h.appidControl, writeControlObject(h.appidControl, typeAppidControl, []uint64{h.appidAcad}))
	add(h.dimstyleControl, writeDimstyleControl(h))
	add(h.nod, writeDictionary(h.nod, 0, nil))
	add(h.layer0, writeLayer(h))
	add(h.styleStandard, writeStyle(h))
	add(h.ltypeByBlock, writeLinetype(h.ltypeByBlock, h.ltypeControl, "ByBlock"))
	add(h.ltypeByLayer, writeLinetype(h.ltypeByLayer, h.ltypeControl, "ByLayer"))
	add(h.ltypeContinuous, writeLinetype(h.ltypeContinuous, h.ltypeControl, "Continuous"))
	add(h.appidAcad, writeAppid(h))
	add(h.vportActive, writeVport(h))
	add(h.dimstyleStandard, writeDimstyle(h))

	first, last := uint64(0), uint64(0)
	if len(ents) > 0 {
		first, last = h.entityBase, h.entityBase+uint64(len(ents))-1
	}
	add(h.modelHdr, writeBlockHeader(h.modelHdr, h.blockControl, "*Model_Space",
		blockHeaderRefs{block: h.modelBlock, endblk: h.modelEndblk, first: first, last: last, layout: h.layoutModel}))
	add(h.paperHdr, writeBlockHeader(h.paperHdr, h.blockControl, "*Paper_Space",
		blockHeaderRefs{block: h.paperBlock, endblk: h.paperEndblk, layout: h.layoutPaper}))
	add(h.modelBlock, writeBlock(h.modelBlock, h.layer0, "*Model_Space"))
	add(h.modelEndblk, writeEndblk(h.modelEndblk, h.layer0))
	add(h.paperBlock, writeBlock(h.paperBlock, h.layer0, "*Paper_Space"))
	add(h.paperEndblk, writeEndblk(h.paperEndblk, h.layer0))

	for i, e := range ents {
		handle := h.entityBase + uint64(i)
		prev, next := uint64(0), uint64(0)
		if i > 0 {
			prev = handle - 1
		}
		if i < len(ents)-1 {
			next = handle + 1
		}
		b, err := encodeModelEntity(handle, e, prev, next, h.layer0)
		if err != nil {
			return nil, nil, err
		}
		add(handle, b)
	}
	return buf, refs, nil
}

// writableEntities filters to the entity types the geometry encoder supports.
func writableEntities(entities []Entity) []Entity {
	var ents []Entity
	for _, e := range entities {
		if _, ok := objectTypeBS(e); ok {
			ents = append(ents, e)
		}
	}
	return ents
}

// writeBlockControl frames the BLOCK_CONTROL object. numentries excludes *Model_Space and
// *Paper_Space, which are appended as hard-owner handles after the (null) owner and xdic.
func writeBlockControl(h *graphHandles) []byte {
	b := newObjectBody(h.blockControl, typeBlockControl)
	b.data.WriteBL(0)                       // numreactors
	b.data.WriteBL(0)                       // numentries (model/paper space not counted)
	b.handles.WriteHandle(softPtrCode, 0)   // owner: root (null)
	b.handles.WriteHandle(hardOwnerCode, 0) // xdicobjhandle (null)
	b.handles.WriteHandle(hardOwnerCode, h.modelHdr)
	b.handles.WriteHandle(hardOwnerCode, h.paperHdr)
	return frameObject(b)
}

// writeLtypeControl frames the LTYPE_CONTROL object. numentries counts only the soft-owner
// linetypes (Continuous); BYLAYER and BYBLOCK follow as hard-owner handles (ODA §20.4.57).
func writeLtypeControl(h *graphHandles) []byte {
	b := newObjectBody(h.ltypeControl, typeLtypeControl)
	b.data.WriteBL(0)                       // numreactors
	b.data.WriteBL(1)                       // numentries: Continuous only
	b.handles.WriteHandle(softPtrCode, 0)   // owner: root (null)
	b.handles.WriteHandle(hardOwnerCode, 0) // xdicobjhandle (null)
	b.handles.WriteHandle(softOwnerCode, h.ltypeContinuous)
	b.handles.WriteHandle(hardOwnerCode, h.ltypeByLayer)
	b.handles.WriteHandle(hardOwnerCode, h.ltypeByBlock)
	return frameObject(b)
}

// writeDimstyleControl frames the DIMSTYLE_CONTROL object. Beyond the generic control
// layout it carries an extra num_morehandles count (a libredwg-observed field, written 0)
// after numentries; the single dimstyle is a soft-owner handle.
func writeDimstyleControl(h *graphHandles) []byte {
	b := newObjectBody(h.dimstyleControl, typeDimstyleControl)
	b.data.WriteBL(0)                                        // numreactors
	b.data.WriteBS(1)                                        // num_entries (BitShort, not BL)
	b.data.WriteRC(1)                                        // num_morehandles (one APPID handle per dimstyle)
	b.handles.WriteHandle(softPtrCode, 0)                    // owner: root (null)
	b.handles.WriteHandle(hardOwnerCode, 0)                  // xdicobjhandle (null)
	b.handles.WriteHandle(softOwnerCode, h.dimstyleStandard) // entry
	b.handles.WriteHandle(hardPtrCode, h.dimstyleStandard)   // morehandle
	return frameObject(b)
}

// writeControlObject frames a generic symbol-table control object (LAYER/STYLE/VIEW/UCS/
// VPORT/APPID CONTROL): the common header, the entry count, then the handle stream — a null
// owner (the root), the xdic, and the entry records as soft-owner handles. BLOCK, LTYPE and
// DIMSTYLE controls have extra fields and are framed by their own writers.
func writeControlObject(handle uint64, typ int, entries []uint64) []byte {
	b := newObjectBody(handle, typ)
	b.data.WriteBL(0)                       // numreactors
	b.data.WriteBL(len(entries))            // numentries
	b.handles.WriteHandle(softPtrCode, 0)   // owner: root (null soft pointer)
	b.handles.WriteHandle(hardOwnerCode, 0) // xdicobjhandle (null)
	for _, e := range entries {
		b.handles.WriteHandle(softOwnerCode, e)
	}
	return frameObject(b)
}
