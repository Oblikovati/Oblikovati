// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// entityRecord is the persisted fingerprint of one entity in a context snapshot:
// its kind and lineage bytes. It carries no live object — it is what a key is
// validated against immediately after reload, before the B-rep is recomputed.
type entityRecord struct {
	kind    EntityKind
	lineage []byte
}

// keyContext is a versioned topology snapshot. While a document is live, source
// enumerates the current entities (rebuilt on each recompute). snapshot is the
// captured state written by SaveContextToArray and restored by LoadContextToArray,
// used to validate keys after reload until the source is re-pointed at the rebuilt
// B-rep (see KeyManager.RebindSource).
//
// index is an O(1) exact-bind acceleration over a [RevisionedSource] (M31-F08): a
// (kind, lineage)→entity map, rebuilt lazily only when the source's revision moves
// or the source is re-pointed. It is absent for a source that does not report a
// revision, which then binds by linear scan exactly as before.
type keyContext struct {
	id       ContextID
	source   EntitySource
	snapshot []entityRecord

	index      map[indexKey]Entity
	indexedRev uint64
	indexed    bool
}

// indexKey is the composite identity an exact bind matches on — the same (kind,
// lineage) pair the linear scan compares, so the index never changes which entity a
// key resolves to.
type indexKey struct {
	kind    EntityKind
	lineage string
}

// lookupExact returns the live entity whose kind and lineage equal the key, or nil.
// It is O(1) when the source reports a revision (consulting a cached index, rebuilt
// only when the revision changes); otherwise it falls back to the linear scan, so an
// un-revisioned source behaves exactly as it did before F08.
func (c *keyContext) lookupExact(k RefKey) Entity {
	if c.source == nil {
		return nil
	}
	rev, ok := revisionOf(c.source)
	if !ok {
		return exactMatch(k, c.source.Entities())
	}
	if !c.indexed || rev != c.indexedRev {
		c.rebuildIndex(rev)
	}
	return c.index[indexKey{kind: k.kind, lineage: string(k.payload)}]
}

// rebuildIndex refreshes the (kind, lineage)→entity map from the current source and
// stamps it with rev. On the rare degenerate topology where two entities share a
// (kind, lineage), the last wins — but lineage is unique by construction, so this
// matches the scan's single-result contract in every real body.
func (c *keyContext) rebuildIndex(rev uint64) {
	ents := c.source.Entities()
	idx := make(map[indexKey]Entity, len(ents))
	for _, e := range ents {
		idx[indexKey{kind: e.EntityKind(), lineage: string(e.Lineage().LineageKey())}] = e
	}
	c.index = idx
	c.indexedRev = rev
	c.indexed = true
}

// invalidateIndex drops the cached index so the next exact bind rebuilds it. Called
// when the source is re-pointed (RebindSource), since a new source may legitimately
// reuse a revision value the old one already had.
func (c *keyContext) invalidateIndex() {
	c.index = nil
	c.indexed = false
}

// revisionOf reports a source's revision when it implements [RevisionedSource].
func revisionOf(s EntitySource) (uint64, bool) {
	rs, ok := s.(RevisionedSource)
	if !ok {
		return 0, false
	}
	return rs.Revision(), true
}

// captureSnapshot records the current source entities into the context snapshot.
func (c *keyContext) captureSnapshot() {
	if c.source == nil {
		return
	}
	entities := c.source.Entities()
	c.snapshot = make([]entityRecord, 0, len(entities))
	for _, e := range entities {
		lineage := append([]byte(nil), e.Lineage().LineageKey()...)
		c.snapshot = append(c.snapshot, entityRecord{kind: e.EntityKind(), lineage: lineage})
	}
}

// encode serializes the context snapshot:
// [id u64][count u32]{ [kind u32][len u32][lineage] }.
func (c *keyContext) encode() []byte {
	buf := make([]byte, 0, 12)
	buf = binary.LittleEndian.AppendUint64(buf, uint64(c.id))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(c.snapshot)))
	for _, rec := range c.snapshot {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(rec.kind))
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(rec.lineage)))
		buf = append(buf, rec.lineage...)
	}
	return buf
}

// decodeContext parses bytes produced by keyContext.encode.
func decodeContext(data []byte) (*keyContext, error) {
	r := bytes.NewReader(data)
	id, err := readUint64(r)
	if err != nil {
		return nil, err
	}
	count, err := readUint32(r)
	if err != nil {
		return nil, err
	}
	c := &keyContext{id: ContextID(id), snapshot: make([]entityRecord, 0, count)}
	for i := uint32(0); i < count; i++ {
		rec, err := readRecord(r)
		if err != nil {
			return nil, err
		}
		c.snapshot = append(c.snapshot, rec)
	}
	return c, nil
}

func readRecord(r *bytes.Reader) (entityRecord, error) {
	kind, err := readUint32(r)
	if err != nil {
		return entityRecord{}, err
	}
	n, err := readUint32(r)
	if err != nil {
		return entityRecord{}, err
	}
	lineage := make([]byte, n)
	if err := readFull(r, lineage); err != nil {
		return entityRecord{}, err
	}
	return entityRecord{kind: EntityKind(kind), lineage: lineage}, nil
}

func readUint64(r *bytes.Reader) (uint64, error) {
	var b [8]byte
	if err := readFull(r, b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b[:]), nil
}

func readUint32(r *bytes.Reader) (uint32, error) {
	var b [4]byte
	if err := readFull(r, b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b[:]), nil
}

// readFull fills dst entirely, turning a short read into a clear truncation error.
func readFull(r *bytes.Reader, dst []byte) error {
	if _, err := io.ReadFull(r, dst); err != nil {
		return fmt.Errorf("identity: truncated context data: %w", err)
	}
	return nil
}
