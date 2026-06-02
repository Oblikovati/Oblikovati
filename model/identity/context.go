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
type keyContext struct {
	id       ContextID
	source   EntitySource
	snapshot []entityRecord
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
