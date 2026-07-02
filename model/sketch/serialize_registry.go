// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "fmt"

// Entity serialization is a registry keyed by [EntityKind]: each kind contributes ONE
// codec pairing its encode and decode closures, registered together in a
// serialize_codecs_*.go init(). This replaces the two hand-maintained switch pairs
// (serializeEntity/restoreEntity and their 3D twins) whose halves nothing linked —
// the same failure class that silently dropped features on reload (#1416) and made
// serialize.go's default: branch a runtime save failure (#1624, audit I1). Now a
// kind without BOTH halves cannot register, and serialize_registry_test asserts the
// registered sets match the persisted vocabulary exactly.
//
// 2D and 3D registries are separate because they encode into different recipe rows
// ([EntityData] vs [Entity3DData]) even where the kind spelling is shared.

// entityCodec is one 2D kind's serialization pair. encode receives the concrete
// entity (asserting its own type, as the switch case it replaces did); decode
// rebuilds it inside a restoring sketch, resolving point ids through the restorer.
type entityCodec struct {
	encode func(e Entity) (EntityData, error)
	decode func(r *sketchRestorer, ed EntityData) (Entity, error)
}

// entityCodec3D is one 3D kind's serialization pair. decode receives the entity's
// defining points already resolved (restoreEntity3D looks them up once for every
// kind, as the switch it replaces did).
type entityCodec3D struct {
	encode func(e Entity) (Entity3DData, error)
	decode func(s *Sketch3D, ed Entity3DData, pts []*Point3D) (Entity, error)
}

// entityCodecs2D / entityCodecs3D map a kind to its codec. Populated by the
// serialize_codecs_*.go init()s; read-only after init, so no synchronization.
var (
	entityCodecs2D = map[EntityKind]entityCodec{}
	entityCodecs3D = map[EntityKind]entityCodec3D{}
)

// registerEntityCodec records one 2D kind's codec, panicking on a duplicate or an
// incomplete pair — a programming error caught at startup, not a silent overwrite.
func registerEntityCodec(kind EntityKind, c entityCodec) {
	mustAcceptCodec("sketch", string(kind), c.encode == nil || c.decode == nil, isRegistered2D(kind))
	entityCodecs2D[kind] = c
}

// registerEntityCodec3D records one 3D kind's codec under the same rules.
func registerEntityCodec3D(kind EntityKind, c entityCodec3D) {
	mustAcceptCodec("sketch 3D", string(kind), c.encode == nil || c.decode == nil, isRegistered3D(kind))
	entityCodecs3D[kind] = c
}

func isRegistered2D(kind EntityKind) bool { _, dup := entityCodecs2D[kind]; return dup }
func isRegistered3D(kind EntityKind) bool { _, dup := entityCodecs3D[kind]; return dup }

// mustAcceptCodec is the shared registration gate: a codec must carry a kind and
// both halves, exactly once.
func mustAcceptCodec(family, kind string, half, dup bool) {
	if kind == "" {
		panic(fmt.Sprintf("%s: registerEntityCodec with empty kind", family))
	}
	if half {
		panic(fmt.Sprintf("%s: codec for kind %q must have both encode and decode", family, kind))
	}
	if dup {
		panic(fmt.Sprintf("%s: duplicate entity codec for kind %q", family, kind))
	}
}

// registeredEntityKinds2D returns the 2D kinds with a codec, for the closure test.
func registeredEntityKinds2D() []EntityKind {
	out := make([]EntityKind, 0, len(entityCodecs2D))
	for k := range entityCodecs2D {
		out = append(out, k)
	}
	return out
}

// registeredEntityKinds3D returns the 3D kinds with a codec, for the closure test.
func registeredEntityKinds3D() []EntityKind {
	out := make([]EntityKind, 0, len(entityCodecs3D))
	for k := range entityCodecs3D {
		out = append(out, k)
	}
	return out
}
