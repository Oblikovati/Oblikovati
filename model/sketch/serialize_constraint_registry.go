// SPDX-License-Identifier: GPL-2.0-only

package sketch

// Constraint serialization is a registry keyed by [ConstraintKind]: each kind
// contributes ONE codec pairing its encode and decode closures, registered
// together in a serialize_codecs_constraints*.go init(). This replaces the two
// hand-maintained switch pairs (serializeConstraint/restoreConstraint and their
// 3D twins) whose halves nothing linked — the drift class that shipped #1574
// (Symmetry enumerable but not creatable) and #1416 (encode half without decode
// half), and that made an Equal3D save fail at runtime (#1625, audit I2). Now a
// kind without BOTH halves cannot register, and the registry closure test
// asserts the registered sets match the persisted vocabulary exactly.
//
// 2D and 3D registries are separate because they encode into different recipe
// rows ([ConstraintData] vs [Constraint3DRow]) even where the kind spelling is
// shared ("coincident" relates two points in both, but "collinear" is two lines
// in 2D and three points in 3D).

// constraintCodec is one 2D kind's serialization pair. encode receives the
// concrete constraint (asserting its own type, as the switch case it replaces
// did) and fills the row WITHOUT its Kind — serializeConstraint stamps that
// from the constraint's own ConstraintKind so row and registry key can never
// drift. decode rebuilds the constraint inside a restoring sketch, resolving
// operand ids through the restorer.
type constraintCodec struct {
	encode func(c Constraint) (ConstraintData, error)
	decode func(r *sketchRestorer, cd ConstraintData) error
}

// constraintCodec3D is one 3D kind's serialization pair. decode receives the
// row's points already resolved (restoreConstraint3D looks them up once for
// every kind) and resolves curve operands against the full entity map.
type constraintCodec3D struct {
	encode func(c Constraint) (Constraint3DRow, error)
	decode func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error
}

// constraintCodecs2D / constraintCodecs3D map a kind to its codec. Populated by
// the serialize_codecs_constraints*.go init()s; read-only after init.
var (
	constraintCodecs2D = map[ConstraintKind]constraintCodec{}
	constraintCodecs3D = map[ConstraintKind]constraintCodec3D{}
)

// registerConstraintCodec records one 2D kind's codec, panicking on a duplicate
// or an incomplete pair — a programming error caught at startup.
func registerConstraintCodec(kind ConstraintKind, c constraintCodec) {
	_, dup := constraintCodecs2D[kind]
	mustAcceptCodec("sketch constraint", string(kind), c.encode == nil || c.decode == nil, dup)
	constraintCodecs2D[kind] = c
}

// registerConstraintCodec3D records one 3D kind's codec under the same rules.
func registerConstraintCodec3D(kind ConstraintKind, c constraintCodec3D) {
	_, dup := constraintCodecs3D[kind]
	mustAcceptCodec("sketch constraint 3D", string(kind), c.encode == nil || c.decode == nil, dup)
	constraintCodecs3D[kind] = c
}

// registeredConstraintKinds2D returns the 2D kinds with a codec, for the
// closure test.
func registeredConstraintKinds2D() []ConstraintKind {
	out := make([]ConstraintKind, 0, len(constraintCodecs2D))
	for k := range constraintCodecs2D {
		out = append(out, k)
	}
	return out
}

// registeredConstraintKinds3D returns the 3D kinds with a codec, for the
// closure test.
func registeredConstraintKinds3D() []ConstraintKind {
	out := make([]ConstraintKind, 0, len(constraintCodecs3D))
	for k := range constraintCodecs3D {
		out = append(out, k)
	}
	return out
}
