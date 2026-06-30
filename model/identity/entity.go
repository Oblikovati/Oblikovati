// SPDX-License-Identifier: GPL-2.0-only

// Package identity implements persistent topological identity — reference keys.
// This is the single most load-bearing kernel mechanism (parametric-cad §7): a
// key minted for a face/edge must re-resolve to "the same" entity after a
// recompute destroys and recreates the B-rep, and after save/close/reopen.
// Topological naming, not pointer identity (architecture core/05).
//
// A [RefKey] is an opaque, serializable value encoding an entity's GENERATIVE
// LINEAGE — the derivation path a rebuild reproduces identically — not an address
// or array index. A [RefKey] captures that lineage (plus parent/anchor fallback
// hints); the live model rebinds it through the tiered [RecoverLost] recovery and
// reports binding loss as health rather than crashing.
//
// identity defines a small topology SEAM — [Entity], [Lineage] — that the kernel's
// B-rep types (kernel/topo Face/Edge/Vertex) implement, plus the [RefKey] value and
// its versioned encoding that persist a reference across save/reopen and recompute.
package identity

import "oblikovati.org/math"

// EntityKind discriminates the topological/model entities a key can name. Values
// are STABLE: they are encoded into persisted keys, so never renumber them.
type EntityKind uint32

const (
	// KindUnknown is the zero value — never minted into a real key.
	KindUnknown EntityKind = 0
	KindFace    EntityKind = 1
	KindEdge    EntityKind = 2
	KindVertex  EntityKind = 3
	KindBody    EntityKind = 4
	// KindFeature/KindParameter cover non-B-rep entities that also need stable
	// identity (GetReferenceKey applies to features and parameters too).
	KindFeature   EntityKind = 10
	KindParameter EntityKind = 11
	// KindDocument names the document itself — the anchor for document-level
	// attribute sets (#155). There is one such key per document; see [DocumentKey].
	KindDocument EntityKind = 20
	// KindExternal anchors to an entity addressed by an opaque EXTERNAL reference
	// key — a kernel/topo reference key (which sits below this package and cannot
	// mint identity keys), as surfaced to add-ins by body.list / model.referenceKeys.
	// The external key is stored verbatim as the payload; see [ExternalKey].
	KindExternal EntityKind = 21
)

// DocumentKey is the single well-known reference key that names the document itself
// (not any entity within it) — the anchor for document-level attribute sets (#155).
// It is fixed (no lineage payload), so it survives recompute and round-trips through
// the attribute codec like any other key.
func DocumentKey() RefKey { return RefKey{kind: KindDocument} }

// ExternalKey anchors attributes to an entity an add-in can name but this package did not mint:
// it wraps the opaque external reference key the add-in received over the wire (a kernel/topo
// reference key from body.list / model.referenceKeys) verbatim as the payload. Equal external
// bytes — the same body/face re-minting the same reference key after a recompute — yield an equal
// RefKey, so the anchored attributes are found again (the per-entity counterpart of [DocumentKey]).
//
// Example: ss := mgr.AttributeSets(identity.ExternalKey(bodyInfo.Key))
func ExternalKey(external []byte) RefKey {
	return RefKey{kind: KindExternal, payload: append([]byte(nil), external...)}
}

// ExternalRef returns the external reference key wrapped by [ExternalKey], and true, when this is
// an external anchor; otherwise (a document or minted key) it returns nil, false. It is the
// inverse used to render an anchor back to the wire target an add-in addressed it by.
func (k RefKey) ExternalRef() ([]byte, bool) {
	if k.kind != KindExternal {
		return nil, false
	}
	return k.payload, true
}

// String returns a stable lowercase name for diagnostics.
func (k EntityKind) String() string {
	switch k {
	case KindFace:
		return "face"
	case KindEdge:
		return "edge"
	case KindVertex:
		return "vertex"
	case KindBody:
		return "body"
	case KindFeature:
		return "feature"
	case KindParameter:
		return "parameter"
	case KindDocument:
		return "document"
	case KindExternal:
		return "external"
	default:
		return "unknown"
	}
}

// Lineage is the generative derivation of an entity: a structural fingerprint a
// rebuild reproduces identically for "the same" entity, even though the entity
// object is freshly allocated. This is the substance of topological naming — keys
// encode lineage, not addresses. kernel/topo entities derive it from their
// feature/derivation history (e.g. "the top cap face of feature Extrude#3").
type Lineage interface {
	// LineageKey returns the stable, serializable bytes identifying the
	// derivation. Two entities are "the same" across rebuilds iff their
	// LineageKey bytes are equal.
	LineageKey() []byte
}

// Entity is the minimal view the key manager needs of a topological/model entity:
// its kind and its lineage. kernel/topo's Face/Edge/Vertex and the feature types
// implement this from M07/M08; tests use fakes.
type Entity interface {
	EntityKind() EntityKind
	Lineage() Lineage
}

// AncestralLineage is an OPTIONAL capability of a [Lineage]: the key of its parent
// derivation (this lineage with its most-specific step removed). The tiered binder
// uses it to recover a reference to a surviving sibling when the exact entity is
// gone — siblings are the entities that share a parent (M31-F06, #1156). A lineage
// with no meaningful parent (a root) need not implement it; such entities then have
// no ancestral fallback and a lost key resolves to Sick as before.
type AncestralLineage interface {
	Lineage
	// ParentKey returns the stable bytes of the parent lineage, or nil when this
	// lineage is a root with no parent to fall back to.
	ParentKey() []byte
}

// AnchoredEntity is an OPTIONAL capability of an [Entity]: a representative point in
// the body's local frame. The binder uses it only as a geometric tie-breaker when
// several surviving siblings share the key's parent lineage (M31-F06). It is never
// used for exact binding; entities that cannot supply a stable point omit it.
type AnchoredEntity interface {
	Entity
	// Anchor returns a representative point and true, or the zero point and false
	// when no stable anchor exists for this entity.
	Anchor() (math.Point3, bool)
}
