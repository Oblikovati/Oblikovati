// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// writeblocks.go encodes the block layer of the graph: the *Model_Space and *Paper_Space
// block records (BLOCK_HEADER) with their BLOCK/ENDBLK markers, and the model-space drawing
// entities themselves. Model-space entities are a doubly linked list (prev/next soft
// pointers) bracketed by the block record's first/last pointers — the R2000 representation
// the oracle confirms (nolinks = 0, layer is a hard pointer, colour 256 = ByLayer).

// colorByLayer / linewtByLayer are the common-entity defaults for a drawing entity: colour
// and lineweight inherited from the layer (the values the oracle reads on real entities).
const (
	colorByLayer  = 256
	linewtByLayer = 0x1d
)

// writeEntityCommon emits the common-entity data stream (entmode 2 = model space, an explicit
// prev/next link list) and the common handle stream (null xdic, prev, next, layer). It is
// shared by drawing entities and the BLOCK/ENDBLK markers, which are also model-space
// entities. prev/next of 0 encode a null soft pointer (list end).
func writeEntityCommon(b *objectBody, prev, next, layer uint64) {
	b.data.WriteBit(0)            // preview_exists
	b.data.WriteBits(2, 2)        // entmode = 2 (model space)
	b.data.WriteBL(0)             // num_reactors
	b.data.WriteBit(0)            // nolinks = 0 → explicit prev/next handles follow
	b.data.WriteBS(colorByLayer)  // colour (ByLayer)
	b.data.WriteBD(1.0)           // ltype_scale
	b.data.WriteBits(0, 2)        // ltype_flags = ByLayer
	b.data.WriteBits(0, 2)        // plotstyle_flags = ByLayer
	b.data.WriteBS(0)             // invisible
	b.data.WriteRC(linewtByLayer) // lineweight (ByLayer)

	b.handles.WriteHandle(hardOwnerCode, 0) // xdicobjhandle (null)
	b.handles.WriteHandle(softPtrCode, prev)
	b.handles.WriteHandle(softPtrCode, next)
	b.handles.WriteHandle(hardPtrCode, layer)
}

// encodeModelEntity frames one model-space drawing entity (line/circle/arc/…): the common
// entity data with its place in the prev/next list, then the type-specific geometry.
func encodeModelEntity(handle uint64, e Entity, prev, next, layer uint64) ([]byte, error) {
	typ, ok := objectTypeBS(e)
	if !ok {
		return nil, fmt.Errorf("dwg: cannot write entity type %s", e.EntityType().Name())
	}
	b := newObjectBody(handle, typ)
	writeEntityCommon(b, prev, next, layer)
	writeGeometry(b.data, e)
	return frameObject(b), nil
}

// writeBlock frames a BLOCK marker entity: the common entity data (no links — it is the
// definition header, not part of the entity list) followed by the block name.
func writeBlock(handle, layer uint64, name string) []byte {
	b := newObjectBody(handle, int(TypeBlock))
	writeEntityCommon(b, 0, 0, layer)
	writeName(b.data, name)
	return frameObject(b)
}

// writeEndblk frames an ENDBLK marker entity: common entity data only.
func writeEndblk(handle, layer uint64) []byte {
	b := newObjectBody(handle, int(TypeEndblk))
	writeEntityCommon(b, 0, 0, layer)
	return frameObject(b)
}

// blockHeaderRefs carries the cross-references a BLOCK_HEADER needs: its BLOCK/ENDBLK
// markers, the first/last entity of its content (0/0 when empty), and its layout object.
type blockHeaderRefs struct {
	block, endblk, first, last, layout uint64
}

// writeBlockHeader frames a *Model_Space or *Paper_Space block record. The data stream holds
// the name and the block flags (all clear for an ordinary loaded block); the handle stream
// links the block control owner, the BLOCK/ENDBLK markers, the first/last content entity and
// the layout. Field order matches the oracle's R2000 decode (no insert-units/explodable, no
// extra null pointer before BLOCK).
//
//nolint:funlen // sequential BLOCK_HEADER field/handle writes in the fixed R2000 order.
func writeBlockHeader(handle, control uint64, name string, r blockHeaderRefs) []byte {
	b := newObjectBody(handle, int(TypeBlockHeader))
	writeRecordCommon(b, control, name)
	b.data.WriteBit(0)            // anonymous
	b.data.WriteBit(0)            // has attrs
	b.data.WriteBit(0)            // blkisxref
	b.data.WriteBit(0)            // xrefoverlaid
	b.data.WriteBit(0)            // xref loaded
	b.data.Write3BD([3]float64{}) // base point
	writeTextEmpty(b.data)        // xref pathname
	b.data.WriteRC(0)             // insert count terminator (0 inserts)
	writeTextEmpty(b.data)        // block description
	b.data.WriteBL(0)             // preview data size (none)

	// writeRecordCommon already wrote owner (block control) + xdic; append the block links.
	b.handles.WriteHandle(hardOwnerCode, r.block)
	b.handles.WriteHandle(softPtrCode, r.first)
	b.handles.WriteHandle(softPtrCode, r.last)
	b.handles.WriteHandle(hardOwnerCode, r.endblk)
	b.handles.WriteHandle(hardPtrCode, r.layout)
	return frameObject(b)
}
