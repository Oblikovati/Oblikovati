// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"strconv"
	"strings"
	"sync/atomic"
)

// EntityKind discriminates topological entities. Values are stable (they prefix a
// reference key, which is persisted): never renumber.
type EntityKind uint8

const (
	KindVertex EntityKind = 1
	KindEdge   EntityKind = 2
	KindFace   EntityKind = 3
	KindLoop   EntityKind = 4
	KindShell  EntityKind = 5
	KindBody   EntityKind = 6
)

// String returns a stable name for diagnostics.
func (k EntityKind) String() string {
	switch k {
	case KindVertex:
		return "vertex"
	case KindEdge:
		return "edge"
	case KindFace:
		return "face"
	case KindLoop:
		return "loop"
	case KindShell:
		return "shell"
	case KindBody:
		return "body"
	case KindWire:
		return "wire"
	default:
		return "unknown"
	}
}

// LineageToken is one step of an entity's derivation: the feature that produced it,
// the role it plays in that feature, and an index disambiguating siblings of the
// same role (e.g. which profile edge).
type LineageToken struct {
	Feature string
	Role    string
	Index   int
}

// Tok is a terse LineageToken constructor.
func Tok(feature, role string, index int) LineageToken {
	return LineageToken{Feature: feature, Role: role, Index: index}
}

// Lineage is the generative derivation path of an entity — the seed of its
// reference key. Two entities are "the same" across rebuilds iff their lineages are
// equal. It is a value type (comparable by its serialized [Lineage.Key]).
//
// A lineage is immutable once built (tokens is unexported and never mutated; [Lineage.Tokens]
// returns a copy), so its serialized key is computed once at construction and memoized in keyStr.
// The cached string is shared by every value copy and is safe to share because strings are
// immutable — this removes the redundant per-call key rebuilds that dominated boolean provenance
// naming on dense imprint sets (#1578).
type Lineage struct {
	tokens []LineageToken
	keyStr string
}

// newLineage is the single internal constructor: it memoizes the serialized key once so all
// callers (and every value copy) get an O(1) [Lineage.Key]/[Lineage.KeyString].
func newLineage(tokens []LineageToken) Lineage {
	return Lineage{tokens: tokens, keyStr: serializeKey(tokens)}
}

// NewLineage builds a lineage from ordered tokens (root first).
func NewLineage(tokens ...LineageToken) Lineage {
	return newLineage(tokens)
}

// Tokens returns the derivation tokens, root first.
func (l Lineage) Tokens() []LineageToken {
	out := make([]LineageToken, len(l.tokens))
	copy(out, l.tokens)
	return out
}

// serializeKey renders the deterministic identity serialization of ordered tokens. The
// separators cannot appear in feature/role names by convention (they are ids), so the encoding is
// unambiguous. Called once per lineage at construction; callers read the memoized result.
func serializeKey(tokens []LineageToken) string {
	var b strings.Builder
	for i, t := range tokens {
		if i > 0 {
			b.WriteByte('/')
		}
		b.WriteString(t.Feature)
		b.WriteByte(':')
		b.WriteString(t.Role)
		b.WriteByte('#')
		b.WriteString(strconv.Itoa(t.Index))
	}
	return b.String()
}

// KeyString returns the memoized serialization (zero-alloc). Prefer it over [Lineage.Key] in hot
// paths that only compare or map by the key. A zero-value Lineage (nil tokens, built by a struct
// literal rather than a constructor) serializes lazily to the empty key.
func (l Lineage) KeyString() string {
	if l.keyStr == "" && len(l.tokens) > 0 {
		return serializeKey(l.tokens) // a literal-built lineage that bypassed the constructor
	}
	return l.keyStr
}

// Key returns the deterministic serialization used for identity comparison, as bytes.
func (l Lineage) Key() []byte { return []byte(l.KeyString()) }

// String renders the lineage for diagnostics.
func (l Lineage) String() string { return l.KeyString() }

// idSeq mints session-stable entity ids (not persisted; identity across sessions is
// the reference key).
var idSeq atomic.Uint64

func nextID() uint64 { return idSeq.Add(1) }
