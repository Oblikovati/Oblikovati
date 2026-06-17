// SPDX-License-Identifier: GPL-2.0-only

package dxf

import "oblikovati.org/kernel/exchange/drawing"

// maxBlockDepth bounds nested-INSERT recursion (a second guard beyond cycle detection).
const maxBlockDepth = 32

// maxExpandedEntities caps the geometry produced by INSERT expansion so a pathological or
// deeply nested file cannot exhaust memory.
const maxExpandedEntities = 4_000_000

// blockSet holds the BLOCKS section's definitions, keyed by a synthetic handle per block
// name (DXF blocks are name-keyed, but the neutral drawing.Insert references a block by a
// uint64 handle, so a name is mapped to a stable synthetic handle). It mirrors the DWG
// decoder's collector: a block's geometry and its own nested INSERTs are stored separately so
// model-space INSERTs can expand them with the right transform.
type blockSet struct {
	nameToHandle map[string]uint64
	next         uint64
	entities     map[uint64][]drawing.Entity  // a block's geometry, by synthetic handle
	inserts      map[uint64][]*drawing.Insert // a block's own INSERTs, by synthetic handle
}

// newBlockSet returns an empty block set (a drawing with no BLOCKS section still expands its
// model INSERTs against this — they simply resolve to no geometry).
func newBlockSet() *blockSet {
	return &blockSet{
		nameToHandle: map[string]uint64{},
		entities:     map[uint64][]drawing.Entity{},
		inserts:      map[uint64][]*drawing.Insert{},
	}
}

// handleFor returns the stable synthetic handle for a block name, assigning one on first use.
func (bs *blockSet) handleFor(name string) uint64 {
	if h, ok := bs.nameToHandle[name]; ok {
		return h
	}
	bs.next++
	bs.nameToHandle[name] = bs.next
	return bs.next
}

// decodeBlocks parses the BLOCKS section into a blockSet. Each BLOCK introduces a named
// definition; the entity groups between its header and the matching ENDBLK are the block's
// geometry (INSERTs among them are nested references). The *Model_Space/*Paper_Space blocks
// are normally empty (their entities live in the ENTITIES section) and add nothing here.
func decodeBlocks(pairs []pair) *blockSet {
	bs := newBlockSet()
	var current string
	inBlock := false
	for _, g := range splitEntities(pairs) {
		switch g.name {
		case "BLOCK":
			current = blockName(g.body)
			inBlock = true
		case "ENDBLK":
			inBlock = false
		default:
			if inBlock {
				bs.addBlockEntity(current, g.name, g.body)
			}
		}
	}
	return bs
}

// blockName reads a BLOCK header's name (code 2).
func blockName(body []pair) string {
	for _, p := range body {
		if p.code == 2 {
			return p.text()
		}
	}
	return ""
}

// addBlockEntity decodes one entity of a block definition into the block set, filing INSERTs
// separately so nested expansion sees them.
func (bs *blockSet) addBlockEntity(block, entType string, body []pair) {
	h := bs.handleFor(block)
	if entType == "INSERT" {
		bs.inserts[h] = append(bs.inserts[h], decodeInsert(indexByCode(body), bs))
		return
	}
	if e, err := decodeEntity(entType, body); err == nil && e != nil {
		bs.entities[h] = append(bs.entities[h], e)
	}
}

// decodeInsert reads an INSERT block reference. Scale factors default to 1, rotation (code
// 50) is degrees converted to the model's radians, and the block name (code 2) resolves to
// the block's synthetic handle.
func decodeInsert(m map[int]pair, bs *blockSet) *drawing.Insert {
	ins := &drawing.Insert{Handle: handleOf(m)}
	if p, ok := m[2]; ok {
		ins.BlockHeader = bs.handleFor(p.text())
	}
	ins.Insertion, _ = coord(m, 10, 20, 30)
	sx, _ := optFloatDefault(m, 41, 1)
	sy, _ := optFloatDefault(m, 42, 1)
	sz, _ := optFloatDefault(m, 43, 1)
	ins.Scale = [3]float64{sx, sy, sz}
	rot, _ := optFloat(m, 50)
	ins.Rotation = degToRad(rot)
	return ins
}

// expandModel returns the final model entity list: the directly-drawn geometry plus every
// model-space INSERT expanded into transformed copies of its block's geometry.
func expandModel(geometry []drawing.Entity, inserts []*drawing.Insert, bs *blockSet) []drawing.Entity {
	out := geometry
	for _, in := range inserts {
		out = bs.expand(in, drawing.IdentityAffine(), out, map[uint64]bool{}, 0)
	}
	return out
}

// expand appends a block reference's geometry to out, transformed by parent∘insert, and
// recurses into the block's own INSERTs. The visiting set breaks reference cycles and depth
// bounds runaway nesting — mirroring the DWG decoder's expansion.
func (bs *blockSet) expand(in *drawing.Insert, parent drawing.Affine, out []drawing.Entity, visiting map[uint64]bool, depth int) []drawing.Entity {
	if depth >= maxBlockDepth || visiting[in.BlockHeader] || len(out) >= maxExpandedEntities {
		return out
	}
	m := parent.Mul(drawing.InsertAffine(in))
	visiting[in.BlockHeader] = true
	for _, e := range bs.entities[in.BlockHeader] {
		out = append(out, drawing.TransformEntity(e, m))
	}
	for _, ni := range bs.inserts[in.BlockHeader] {
		out = bs.expand(ni, m, out, visiting, depth+1)
	}
	delete(visiting, in.BlockHeader)
	return out
}
