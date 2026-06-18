// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"crypto/sha1" //nolint:gosec // SHA-1 is mandated by RFC 4122 §4.3 for v5 names, not security
	"encoding/hex"
	"fmt"
	"strings"
)

// Persistent sketch references (#153): a sketch or sketch entity is named durably by a
// UUID DERIVED from its document's stable GUID and its document-local id, rather than by
// the transient session id (which is re-minted on load). Deriving from the document GUID
// makes a cross-document collision structurally impossible: two documents never share a
// GUID, so the same local id in two files yields two different UUIDs. Within one document
// the local id is unique, so the UUID is unique there too. Because the derivation is a
// pure function of (GUID, kind, local id) and the local id now survives the .obk round
// trip, the same entity yields the same UUID across save/load and unrelated edits.

// Sketch-reference kind tags. They namespace the derived UUID so a sketch and an entity
// that happened to share a local id (they cannot today, but the tag future-proofs it)
// can never collide, and so the kind is self-describing in the derived name.
const (
	sketchKindTag       = "sketch"
	sketchEntityKindTag = "sketchEntity"
)

// SketchKey returns the persistent reference UUID of a sketch, derived from its document's
// GUID and the sketch's document-local id.
//
//	key, err := identity.SketchKey(doc.FileIdentity().InternalName, uint64(sk.ID()))
func SketchKey(documentGUID string, localID uint64) (string, error) {
	return deriveDocumentUUID(documentGUID, sketchKindTag, localID)
}

// SketchEntityKey returns the persistent reference UUID of a sketch entity (line, point,
// arc, …), derived from its document's GUID and the entity's document-local id.
//
//	key, err := identity.SketchEntityKey(doc.FileIdentity().InternalName, uint64(e.EntityID()))
func SketchEntityKey(documentGUID string, localID uint64) (string, error) {
	return deriveDocumentUUID(documentGUID, sketchEntityKindTag, localID)
}

// deriveDocumentUUID builds an RFC 4122 version-5 (name-based, SHA-1) UUID whose namespace
// is the document GUID and whose name is "<kind>:<localID>". The document GUID is the
// namespace, so the result is unique per document by construction.
func deriveDocumentUUID(documentGUID, kind string, localID uint64) (string, error) {
	ns, err := parseUUID(documentGUID)
	if err != nil {
		return "", fmt.Errorf("identity: derive %s uuid: document guid %q: %w", kind, documentGUID, err)
	}
	name := fmt.Sprintf("%s:%d", kind, localID)
	return formatUUID(uuidV5(ns, name)), nil
}

// uuidV5 hashes the namespace bytes followed by the name (RFC 4122 §4.3) and stamps the
// version (5) and variant (RFC 4122) bits.
func uuidV5(namespace [16]byte, name string) [16]byte {
	h := sha1.New() //nolint:gosec // see file header: RFC 4122 v5 is defined over SHA-1
	h.Write(namespace[:])
	h.Write([]byte(name))
	sum := h.Sum(nil)

	var out [16]byte
	copy(out[:], sum[:16])
	out[6] = (out[6] & 0x0f) | 0x50 // version 5
	out[8] = (out[8] & 0x3f) | 0x80 // RFC 4122 variant
	return out
}

// parseUUID decodes the canonical 8-4-4-4-12 hex form into 16 bytes.
func parseUUID(s string) ([16]byte, error) {
	var out [16]byte
	clean := strings.ReplaceAll(s, "-", "")
	if len(clean) != 32 {
		return out, fmt.Errorf("want 32 hex digits (8-4-4-4-12), got %d in %q", len(clean), s)
	}
	b, err := hex.DecodeString(clean)
	if err != nil {
		return out, fmt.Errorf("not hexadecimal: %w", err)
	}
	copy(out[:], b)
	return out, nil
}

// formatUUID renders 16 bytes as the canonical 8-4-4-4-12 lowercase hex string.
func formatUUID(b [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
