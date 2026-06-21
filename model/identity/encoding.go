// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// The reference-key wire format is VERSIONED so the lineage-naming scheme can
// evolve (F03–F05 and beyond) without orphaning saved documents (M31-F07, #1157).
// FreeCAD's hard-won lesson: a topological-naming scheme must be versioned and
// migratable. A key is self-describing — it carries the scheme that minted it — and
// pre-M31 keys (which have no version tag) are still decoded, so old files reopen.
//
// Versioned (v1+) envelope, fixed little-endian layout (architecture core/05):
//
//	[magic "OKRK"][scheme u8][ctx u64][kind u32][payloadLen u32][payload]
//
// Legacy (pre-M31, "v0") envelope — no magic, the bytes begin with the context id:
//
//	[ctx u64][kind u32][payloadLen u32][payload]
//
// Only the identity triple (ctx, kind, lineage payload) is encoded. The F06 fallback
// hints (parent lineage, geometric anchor) are deliberately NOT serialized: the
// encoding is the attribute-anchor map key (model/attr), and that contract requires
// equal-lineage keys to encode identically across a recompute. The anchor is a float
// that can drift on recompute, so folding it into the identity bytes would silently
// orphan a face's attributes — the very failure topological naming exists to prevent.
var keyMagic = [4]byte{'O', 'K', 'R', 'K'}

// SchemeLegacy is the implicit version of pre-M31 keys, which carry no version tag.
// SchemeCurrent is what newly minted keys are stamped with; bump it whenever the
// payload's meaning changes and add a decode path for the previous value.
const (
	SchemeLegacy  uint8 = 0
	SchemeCurrent uint8 = 1
)

// Encode serializes the key to its self-describing, versioned envelope. Newly
// minted keys are always written at SchemeCurrent, so reopening and re-saving an
// old document upgrades its keys in place.
//
// Example: persisted := key.Encode(); ... ; restored, _ := DecodeKey(persisted).
func (k RefKey) Encode() []byte {
	buf := make([]byte, 0, len(keyMagic)+17+len(k.payload))
	buf = append(buf, keyMagic[:]...)
	buf = append(buf, SchemeCurrent)
	buf = binary.LittleEndian.AppendUint64(buf, uint64(k.ctx))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(k.kind))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(k.payload)))
	return append(buf, k.payload...)
}

// DecodeKey parses bytes produced by [RefKey.Encode], transparently migrating a
// pre-M31 (legacy, unversioned) key — so old documents rebind rather than orphan.
// The decoded key's Scheme reports which envelope it came from.
func DecodeKey(data []byte) (RefKey, error) {
	if hasMagic(data) {
		return decodeVersioned(data)
	}
	return decodeLegacy(data)
}

// hasMagic reports whether data begins with the versioned-envelope marker. A legacy
// key begins with its context id, which cannot collide: it would require a manager
// to mint over 1.2 billion contexts AND the following byte to be a valid scheme, and
// decodeVersioned still length-validates, so a false positive cannot bind wrongly.
func hasMagic(data []byte) bool {
	return len(data) >= len(keyMagic) && bytes.Equal(data[:len(keyMagic)], keyMagic[:])
}

// decodeVersioned parses the v1+ envelope: magic, scheme, then the identity triple.
func decodeVersioned(data []byte) (RefKey, error) {
	const headerLen = 5 + 16 // magic+scheme, then ctx+kind+payloadLen
	if len(data) < headerLen {
		return RefKey{}, fmt.Errorf("identity: versioned key too short: %d bytes, want >= %d", len(data), headerLen)
	}
	scheme := data[len(keyMagic)]
	if scheme == SchemeLegacy || scheme > SchemeCurrent {
		return RefKey{}, fmt.Errorf("identity: unknown key scheme %d, this build understands 1..%d", scheme, SchemeCurrent)
	}
	body := data[len(keyMagic)+1:]
	k, err := decodeTriple(body)
	if err != nil {
		return RefKey{}, err
	}
	k.scheme = scheme
	return k, nil
}

// decodeLegacy parses a pre-M31 key (no magic, scheme assumed SchemeLegacy).
func decodeLegacy(data []byte) (RefKey, error) {
	k, err := decodeTriple(data)
	if err != nil {
		return RefKey{}, err
	}
	k.scheme = SchemeLegacy
	return k, nil
}

// decodeTriple parses the shared identity body [ctx u64][kind u32][len u32][payload],
// validating that the declared payload length consumes exactly the trailing bytes.
func decodeTriple(b []byte) (RefKey, error) {
	if len(b) < 16 {
		return RefKey{}, fmt.Errorf("identity: key body too short: %d bytes, want >= 16", len(b))
	}
	ctx := ContextID(binary.LittleEndian.Uint64(b[0:8]))
	kind := EntityKind(binary.LittleEndian.Uint32(b[8:12]))
	n := binary.LittleEndian.Uint32(b[12:16])
	if int(n) != len(b)-16 {
		return RefKey{}, fmt.Errorf("identity: key payload length %d does not match %d trailing bytes", n, len(b)-16)
	}
	payload := make([]byte, n)
	copy(payload, b[16:])
	return RefKey{ctx: ctx, kind: kind, payload: payload}, nil
}
