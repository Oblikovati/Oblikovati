// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/internal/collview"
)

// The DS-joint surface (M12-F02): the degrees-of-freedom / imposed-motion view of a joint
// that motion and simulation consumers (M18) read. A DS joint of a kind exposes its DOF set
// (translational/rotational), each with an imposed-motion mode (free/driven/locked). For F02
// it is the kinematic description; full imposed-motion enforcement during a sweep is the
// drive layer (M12-F03). A locked DOF reduces the joint's reported free DOF.

// dsDOF is one degree of freedom of a DS joint.
type dsDOF struct {
	rotational bool
	imposed    types.DSDOFImposedMotionType
	value      float64
}

// Rotational reports whether this DOF is rotational (false ⇒ translational).
func (d *dsDOF) Rotational() bool { return d.rotational }

// ImposedMotion reports how the DOF's motion is imposed.
func (d *dsDOF) ImposedMotion() types.DSDOFImposedMotionType { return d.imposed }

// Value is the DOF's current value (cm or radians).
func (d *dsDOF) Value() float64 { return d.value }

// dsJoint is a DS joint: a kind, its two origins, and its per-DOF imposed-motion set.
type dsJoint struct {
	id   uint64
	name string
	kind types.DSJointType
	a, b anchor
	dofs []*dsDOF
}

// ID returns the DS joint's session id.
func (j *dsJoint) ID() uint64 { return j.id }

// Type returns the DS joint kind.
func (j *dsJoint) Type() types.DSJointType { return j.kind }

// Name returns the DS joint's display name.
func (j *dsJoint) Name() string { return j.name }

// DOFCount returns the number of degrees of freedom.
func (j *dsJoint) DOFCount() int { return len(j.dofs) }

// DOF returns the i-th degree of freedom, or nil when out of range.
func (j *dsJoint) DOF(i int) contract.DSDegreesOfFreedom {
	return collview.ItemAs(j.dofs, i, func(d *dsDOF) contract.DSDegreesOfFreedom { return d })
}

// FreeDegreesOfFreedom returns the count of DOF that are not locked — the joint's effective
// remaining freedom (driven counts as free for the static view).
func (j *dsJoint) FreeDegreesOfFreedom() int {
	n := 0
	for _, d := range j.dofs {
		if d.imposed != types.DSDOFLocked {
			n++
		}
	}
	return n
}

// dsJointDefinition is the read surface of a DS joint's definition (its kind).
type dsJointDefinition struct{ kind types.DSJointType }

// DSJointType is the kind of DS joint this definition builds.
func (d dsJointDefinition) DSJointType() types.DSJointType { return d.kind }

// dsJointDOFs builds the default (all-free) DOF set for a DS joint kind: the translational
// then rotational freedoms it carries.
func dsJointDOFs(kind types.DSJointType) []*dsDOF {
	trans, rot := dsJointFreedoms(kind)
	dofs := make([]*dsDOF, 0, trans+rot)
	for range trans {
		dofs = append(dofs, &dsDOF{rotational: false, imposed: types.DSDOFFree})
	}
	for range rot {
		dofs = append(dofs, &dsDOF{rotational: true, imposed: types.DSDOFFree})
	}
	return dofs
}

// dsJointFreedoms returns the translational and rotational DOF counts of a DS joint kind.
func dsJointFreedoms(kind types.DSJointType) (translational, rotational int) {
	switch kind {
	case types.DSJointRotational:
		return 0, 1
	case types.DSJointPrismatic:
		return 1, 0
	case types.DSJointCylindrical:
		return 1, 1
	case types.DSJointPlanar:
		return 2, 1
	case types.DSJointSpherical:
		return 0, 3
	default: // rigid / unknown
		return 0, 0
	}
}

// DSJointSet is an assembly's DS-joint collection (the reference API's DSJoints).
type DSJointSet struct {
	items  []*dsJoint
	nextID uint64
	counts map[types.DSJointType]int
}

// NewDSJointSet builds an empty DS-joint set.
func NewDSJointSet() *DSJointSet {
	return &DSJointSet{counts: map[types.DSJointType]int{}}
}

// Count returns the number of DS joints.
func (s *DSJointSet) Count() int { return len(s.items) }

// Item returns the DS joint at index i, or nil when out of range.
func (s *DSJointSet) Item(i int) contract.DSJoint {
	return collview.ItemAs(s.items, i, func(j *dsJoint) contract.DSJoint { return j })
}

// All returns the DS joints in creation order.
func (s *DSJointSet) All() []*dsJoint {
	out := make([]*dsJoint, len(s.items))
	copy(out, s.items)
	return out
}

// ByID returns the DS joint with the given id, or nil.
func (s *DSJointSet) ByID(id uint64) *dsJoint {
	for _, j := range s.items {
		if j.id == id {
			return j
		}
	}
	return nil
}

// Add adds a DS joint of the given kind between two origins.
func (s *DSJointSet) Add(kind types.DSJointType, a, b Ref) *dsJoint {
	s.nextID++
	s.counts[kind]++
	j := &dsJoint{
		id:   s.nextID,
		name: fmt.Sprintf("%s:%d", dsJointName(kind), s.counts[kind]),
		kind: kind,
		a:    toAnchor(a),
		b:    toAnchor(b),
		dofs: dsJointDOFs(kind),
	}
	s.items = append(s.items, j)
	return j
}

// SetImposedMotion sets the imposed motion and value of the DS joint's DOF at dofIndex.
func (s *DSJointSet) SetImposedMotion(id uint64, dofIndex int, imposed types.DSDOFImposedMotionType, value float64) error {
	j := s.ByID(id)
	if j == nil {
		return fmt.Errorf("assembly: no DS joint with id %d", id)
	}
	if dofIndex < 0 || dofIndex >= len(j.dofs) {
		return fmt.Errorf("assembly: DS joint %d has no DOF at index %d (count %d)", id, dofIndex, len(j.dofs))
	}
	j.dofs[dofIndex].imposed = imposed
	j.dofs[dofIndex].value = value
	return nil
}

// Delete removes the DS joint with the given id, returning whether it was found.
func (s *DSJointSet) Delete(id uint64) bool {
	for i, j := range s.items {
		if j.id != id {
			continue
		}
		s.items = append(s.items[:i], s.items[i+1:]...)
		return true
	}
	return false
}

// dsJointName returns a DS joint kind's display prefix.
func dsJointName(kind types.DSJointType) string {
	switch kind {
	case types.DSJointRigid:
		return "DSRigid"
	case types.DSJointRotational:
		return "DSRotational"
	case types.DSJointPrismatic:
		return "DSPrismatic"
	case types.DSJointCylindrical:
		return "DSCylindrical"
	case types.DSJointPlanar:
		return "DSPlanar"
	case types.DSJointSpherical:
		return "DSSpherical"
	default:
		return "DSJoint"
	}
}
