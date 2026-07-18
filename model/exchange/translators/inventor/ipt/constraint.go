// SPDX-License-Identifier: GPL-2.0-only

package ipt

// Assembly constraint decoding. A placement constraint (mate / flush / angle / …) is an
// AmDcSegment node (InventorLoader Read_4E86F047) that positions two occurrences by relating
// a geometry entity on each. Its geometry selections are the parametric intent; the solved
// positions they produce are already captured by the occurrence placement transforms
// (AmRxSegment), so a constrained assembly's GEOMETRY translates correctly from the
// transforms alone. This decodes the constraint KIND (the value a reconstruction of the
// relationship would need first); resolving each selection to a geometry primitive is the
// remaining step for rebuilding the constraint objects themselves.

// ConstraintKind is an assembly placement-constraint type (the node's style byte).
type ConstraintKind uint8

const (
	ConstraintMate     ConstraintKind = 1
	ConstraintFlush    ConstraintKind = 2
	ConstraintAngle    ConstraintKind = 3
	ConstraintSymmetry ConstraintKind = 14
)

// constraintNodeType is the AmDcSegment placement-constraint node (Read_4E86F047);
// constraintStyleOffset is where its style byte sits in the payload for the 2027 layout:
// Header0(6) + ChildRef(4) + flags(4) + skip(4, >2023) + CrossRef(4) + s32(4) + CrossRef(4).
const (
	constraintNodeType    = 0x4E86F047
	constraintStyleOffset = 34
)

// String names the constraint kind (falling back to its numeric style for unmapped values).
func (k ConstraintKind) String() string {
	switch k {
	case ConstraintMate:
		return "mate"
	case ConstraintFlush:
		return "flush"
	case ConstraintAngle:
		return "angle"
	case ConstraintSymmetry:
		return "symmetry"
	default:
		return "constraint"
	}
}

// DecodeConstraintKinds returns the kind of each assembly placement constraint, in node
// order. The full geometry (which entity on which occurrence, and the offset) is not
// resolved — see the package note.
func DecodeConstraintKinds(d *Document) []ConstraintKind {
	var kinds []ConstraintKind
	d.walkSegment("AmDcSegment", func(typ uint32, pay []byte) bool {
		if typ == constraintNodeType && len(pay) > constraintStyleOffset {
			kinds = append(kinds, ConstraintKind(pay[constraintStyleOffset]))
		}
		return true
	})
	return kinds
}
