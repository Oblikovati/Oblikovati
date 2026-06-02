// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

// ContextID identifies a key context (a versioned topology snapshot) within a
// [KeyManager]. It is part of a key's identity and is preserved across save/load.
type ContextID uint64

// RefKey is an opaque, serializable reference to a topological/model entity. It is
// a value type (no pointers), so it can be stored in a feature definition, a
// document descriptor, or a persisted stream and rebound later. Callers treat it
// as opaque; only the [KeyManager] interprets the payload (the entity's lineage).
type RefKey struct {
	ctx     ContextID
	kind    EntityKind
	payload []byte // the entity's LineageKey at mint time
}

// Context returns the key's owning context id.
func (k RefKey) Context() ContextID { return k.ctx }

// Kind returns the kind of entity the key names.
func (k RefKey) Kind() EntityKind { return k.kind }

// IsZero reports whether the key is the zero value (names nothing).
func (k RefKey) IsZero() bool {
	return k.ctx == 0 && k.kind == KindUnknown && len(k.payload) == 0
}

// Equal reports whether two keys are identical (same context, kind, and lineage).
func (k RefKey) Equal(other RefKey) bool {
	return k.ctx == other.ctx && k.kind == other.kind && bytes.Equal(k.payload, other.payload)
}

// MatchType classifies a bind attempt. It modernizes COM ReferenceKeyMatchType,
// trimmed to what the lineage-based design needs.
type MatchType uint8

const (
	// MatchNone: nothing in the current topology matches the key — the reference
	// is lost (a legitimate, non-fatal outcome).
	MatchNone MatchType = iota
	// MatchExact: a single entity with the same kind and lineage was found.
	MatchExact
)

// String returns a stable name for diagnostics.
func (m MatchType) String() string {
	switch m {
	case MatchExact:
		return "exact"
	default:
		return "none"
	}
}

// Encode serializes the key to bytes: [ctx u64][kind u32][len u32][payload]. The
// layout is fixed and little-endian so keys persist portably (architecture core/05).
func (k RefKey) Encode() []byte {
	buf := make([]byte, 0, 16+len(k.payload))
	buf = binary.LittleEndian.AppendUint64(buf, uint64(k.ctx))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(k.kind))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(k.payload)))
	return append(buf, k.payload...)
}

// DecodeKey parses bytes produced by [RefKey.Encode].
func DecodeKey(data []byte) (RefKey, error) {
	if len(data) < 16 {
		return RefKey{}, fmt.Errorf("identity: key too short: %d bytes, want >= 16", len(data))
	}
	ctx := ContextID(binary.LittleEndian.Uint64(data[0:8]))
	kind := EntityKind(binary.LittleEndian.Uint32(data[8:12]))
	n := binary.LittleEndian.Uint32(data[12:16])
	if int(n) != len(data)-16 {
		return RefKey{}, fmt.Errorf("identity: key payload length %d does not match %d trailing bytes", n, len(data)-16)
	}
	payload := make([]byte, n)
	copy(payload, data[16:])
	return RefKey{ctx: ctx, kind: kind, payload: payload}, nil
}

// KeyToString renders a key as a URL-safe base64 string for transport and UI.
func KeyToString(k RefKey) string {
	return base64.URLEncoding.EncodeToString(k.Encode())
}

// StringToKey parses a key from its [KeyToString] form.
func StringToKey(s string) (RefKey, error) {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return RefKey{}, fmt.Errorf("identity: invalid key string: %w", err)
	}
	return DecodeKey(data)
}
