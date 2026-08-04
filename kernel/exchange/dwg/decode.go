// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"fmt"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/drawing"
)

// Decode parses a DWG file end to end and returns its model-space geometry: the curve
// entities (LINE/CIRCLE/ARC/POINT/ELLIPSE/LWPOLYLINE/SPLINE) drawn directly in model
// space, plus the geometry placed by model-space block references (INSERT), expanded in
// place with each insert's transform. Paper-space and block-definition geometry are not
// returned directly — block definitions only appear through the INSERTs that place them
// (ODA entmode; block table). Objects whose type has no geometry decoder yet, or that
// fail to decode, are skipped so one bad record never sinks the import; Warnings record
// what was dropped.
//
// Example:
//
//	dr, warns, err := dwg.Decode(bytes)
//	for _, e := range dr.Entities { /* convert to sketch geometry */ }
func Decode(data []byte) (*Drawing, []string, error) {
	return DecodeWithProgress(data, exchange.TranslationOptions{})
}

// DecodeWithProgress is [Decode] threaded through the shared progress/cancel seam (#1647): opts
// reports one tick per object and aborts the import when its ProgressFunc returns cancel (the
// returned error wraps [exchange.ErrCancelled]). Decode is this call with a zero options value.
func DecodeWithProgress(data []byte, opts exchange.TranslationOptions) (*Drawing, []string, error) {
	h, err := ParseFileHeader(data)
	if err != nil {
		return nil, nil, err
	}
	omb, err := h.ObjectMapBytes(data)
	if err != nil {
		return nil, nil, fmt.Errorf("dwg: object map: %w", err)
	}
	od, err := h.ObjectData(data)
	if err != nil {
		return nil, nil, fmt.Errorf("dwg: object data: %w", err)
	}
	refs, err := parseObjectMap(omb)
	if err != nil {
		return nil, nil, err
	}
	c := &collector{data: od, version: h.Version, blockEntities: map[uint64][]Entity{}, blockInserts: map[uint64][]*Insert{}}
	warns, err := c.collect(refs, opts)
	if err != nil {
		return nil, warns, err
	}
	dr := &Drawing{Entities: c.resolve(), Styles: c.styles}
	return dr, applyHeaderUnits(dr, h, data, warns), nil
}

// applyHeaderUnits reads the drawing's INSUNITS from the header section onto dr; a header that fails
// to parse is non-fatal (the drawing is left unitless and the importer falls back to document
// units), the parse error appended as a warning.
func applyHeaderUnits(dr *Drawing, h *FileHeader, data []byte, warns []string) []string {
	sec, herr := h.HeaderSection(data)
	if herr != nil {
		return warns
	}
	hv, perr := ParseHeaderVars(sec, h.Version)
	if perr != nil {
		return append(warns, perr.Error())
	}
	dr.Units = hv.INSUNITS
	return warns
}

// collector accumulates a drawing's geometry during the object pass, classifying each
// entity by space (entmode): model space is kept directly, paper space is dropped, and
// block-definition geometry is grouped by its owning block so model-space INSERTs can
// expand it (see resolve).
type collector struct {
	data          []byte
	version       Version
	modelEntities []Entity
	modelInserts  []*Insert
	blockEntities map[uint64][]Entity // by owning BLOCK_HEADER handle
	blockInserts  map[uint64][]*Insert
	paperCurves   int // paper-space curve entities skipped (for classification accounting)
	// styles is each entity's own formatting, keyed by its handle (#2015). The decoder always
	// read the colour and line weight to keep the bit stream aligned and threw them away; they
	// are what an imported drawing needs to keep its appearance.
	styles map[uint64]drawing.Style
	// Readers reused across the per-object loop so each object does not allocate a fresh
	// *BitReader: geomReader walks the data stream (held by the cursor), handleReader the
	// handle stream. The loop fully decodes one object before the next, so reuse is safe.
	geomReader   BitReader
	handleReader BitReader
}

// collect walks every referenced object, decoding the geometry and INSERT records and
// sorting them into model space vs block definitions. Per-object failures are collected
// as warnings rather than aborting the drawing.
func (c *collector) collect(refs []ObjectRef, opts exchange.TranslationOptions) ([]string, error) {
	var warns []string
	for i, ref := range refs {
		if err := opts.Report("entities", i, len(refs)); err != nil {
			return warns, err // #1647: honour a cancel between objects
		}
		warns = append(warns, c.classifyRef(ref)...)
	}
	return warns, nil
}

// classifyRef decodes one referenced object and files it under model space or its owning block,
// returning any per-object warnings (a bad object is skipped, not fatal). Paper-space geometry is
// dropped (counted for accounting); a type with no geometry decoder is ignored.
func (c *collector) classifyRef(ref ObjectRef) []string {
	hdr, err := decodeObjectHeader(c.data, ref, c.version)
	if err != nil {
		return nil
	}
	if hdr.Type != TypeInsert && !hdr.Type.IsSketchGeometry() {
		return nil
	}
	cur, err := seekEntity(&c.geomReader, c.data, ref, c.version)
	if err != nil {
		return []string{err.Error()}
	}
	if cur.common.entmode == entmodePaperSpace {
		if hdr.Type.IsSketchGeometry() {
			c.paperCurves++
		}
		return nil
	}
	if w := c.addObject(&cur, hdr); w != "" {
		return []string{w}
	}
	return nil
}

// addObject decodes one classified object (INSERT or curve) and files it under model
// space or its owning block. It returns a warning string on a decode failure.
func (c *collector) addObject(cur *entityCursor, hdr ObjectHeader) string {
	if hdr.Type == TypeInsert {
		in, owner, err := decodeInsert(&c.handleReader, c.data, cur, hdr.Handle, c.version)
		if err != nil {
			return err.Error()
		}
		if cur.common.entmode == entmodeBlock {
			c.blockInserts[owner] = append(c.blockInserts[owner], in)
		} else {
			c.modelInserts = append(c.modelInserts, in)
		}
		return ""
	}
	c.recordStyle(hdr.Handle, cur.common) // per-entity colour / line weight (#2015)
	e, err := decodeEntity(cur.geom, hdr, c.version)
	if err != nil {
		return err.Error()
	}
	if e == nil {
		return ""
	}
	if cur.common.entmode == entmodeBlock {
		owner := commonEntityHandles(&c.handleReader, c.data, cur, c.version)
		c.blockEntities[owner] = append(c.blockEntities[owner], e)
	} else {
		c.modelEntities = append(c.modelEntities, e)
	}
	return ""
}

// entmode values (ODA common_entity_data): which space an entity belongs to.
const (
	entmodeBlock      = 0 // owned by a block definition (owner handle in the handle stream)
	entmodePaperSpace = 1
	entmodeModelSpace = 2
)

// maxExpandedEntities caps the geometry produced by INSERT expansion so a pathological
// or deeply nested file cannot exhaust memory; reaching it stops further expansion.
const maxExpandedEntities = 4_000_000

// maxBlockDepth bounds nested-insert recursion as a second guard beyond cycle detection.
const maxBlockDepth = 32

// resolve produces the final model-space entity list: the directly-drawn entities plus
// every model-space INSERT expanded into transformed copies of its block's geometry.
func (c *collector) resolve() []Entity {
	out := c.modelEntities
	for _, in := range c.modelInserts {
		out = c.expand(in, drawing.IdentityAffine(), out, map[uint64]bool{}, 0)
	}
	return out
}

// expand appends a block reference's geometry to out, transformed by parent∘insert, and
// recurses into the block's own inserts. The visiting set breaks reference cycles and
// depth bounds runaway nesting.
func (c *collector) expand(in *Insert, parent drawing.Affine, out []Entity, visiting map[uint64]bool, depth int) []Entity {
	if depth >= maxBlockDepth || visiting[in.BlockHeader] || len(out) >= maxExpandedEntities {
		return out
	}
	m := parent.Mul(drawing.InsertAffine(in))
	visiting[in.BlockHeader] = true
	for _, e := range c.blockEntities[in.BlockHeader] {
		out = append(out, drawing.TransformEntity(e, m))
	}
	for _, ni := range c.blockInserts[in.BlockHeader] {
		out = c.expand(ni, m, out, visiting, depth+1)
	}
	delete(visiting, in.BlockHeader)
	return out
}

// recordStyle files one entity's own formatting under its handle. An entity that inherits
// everything is not recorded, so a drawing of ordinary geometry carries no style table at all.
func (c *collector) recordStyle(handle uint64, ce commonEntity) {
	s := drawing.Style{
		Color:      ce.colorIndex,
		LineWeight: ce.lineWeight,
		LineType:   dwgLineTypeName(ce.ltypeFlags),
	}
	if s.Color == dwgColorByLayer && s.LineWeight == dwgLineWeightByLayer && s.LineType == "" {
		return
	}
	if c.styles == nil {
		c.styles = map[uint64]drawing.Style{}
	}
	c.styles[handle] = s
}

// dwgLineTypeName maps the two-bit line-type flag onto a record name. Flag 3 names a line-type
// object by handle, which needs the LTYPE records this decoder does not read yet, so it inherits
// rather than guessing a pattern.
func dwgLineTypeName(flags int) string {
	if flags == 2 {
		return "CONTINUOUS"
	}
	return ""
}
