// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/internal/collview"
	"oblikovati.org/model/occurrence"
)

// JointListener is notified when the joint set changes, so the host can raise the assembly's
// joint events (model/compdef wires this to its event bus). Injected, never imported.
type JointListener interface {
	// JointAdded reports a joint just added to the set.
	JointAdded(j contract.AssemblyJoint)
	// JointDeleted reports a joint just removed from the set.
	JointDeleted(j contract.AssemblyJoint)
}

// jointLimits bounds a joint's driven values — a linear range and an angular range, each
// reusing the F01 limits value. It is the host implementation of contract.JointLimits.
type jointLimits struct {
	linear  *limits
	angular *limits
}

// NewJointLimits builds joint limits from optional linear and angular bounds (each side
// included only when its has flag is set); it returns nil when no bound is set (an unbounded
// joint carries no limits object).
func NewJointLimits(linMin float64, hasLinMin bool, linMax float64, hasLinMax bool, angMin float64, hasAngMin bool, angMax float64, hasAngMax bool) *jointLimits {
	linear := NewLimits(linMin, hasLinMin, linMax, hasLinMax, 0, false)
	angular := NewLimits(angMin, hasAngMin, angMax, hasAngMax, 0, false)
	if linear == nil && angular == nil {
		return nil
	}
	return &jointLimits{linear: linear, angular: angular}
}

// LinearMinimum returns the lower linear bound and whether it is set.
func (l *jointLimits) LinearMinimum() (float64, bool) { return rangeBound(l.linear, true) }

// LinearMaximum returns the upper linear bound and whether it is set.
func (l *jointLimits) LinearMaximum() (float64, bool) { return rangeBound(l.linear, false) }

// AngularMinimum returns the lower angular bound and whether it is set.
func (l *jointLimits) AngularMinimum() (float64, bool) { return rangeBound(l.angular, true) }

// AngularMaximum returns the upper angular bound and whether it is set.
func (l *jointLimits) AngularMaximum() (float64, bool) { return rangeBound(l.angular, false) }

// rangeBound reads the min (lower=true) or max bound of an optional limits range.
func rangeBound(lim *limits, lower bool) (float64, bool) {
	if lim == nil {
		return 0, false
	}
	if lower {
		return lim.Minimum()
	}
	return lim.Maximum()
}

// JointSet is an assembly's joint collection (the reference API's AssemblyJoints, plus
// Add{Rigid,Rotational,…} factories). Like ConstraintSet it owns no geometry; its joints
// emit residual bundles that the assembly solve consumes alongside the constraints.
type JointSet struct {
	occs     *occurrence.Occurrences
	items    []Joint
	listener JointListener
	nextID   uint64
	counts   map[types.AssemblyJointType]int
}

// NewJointSet builds an empty joint set over the given occurrences. The listener (may be
// nil) receives add/delete notifications.
func NewJointSet(occs *occurrence.Occurrences, l JointListener) *JointSet {
	return &JointSet{occs: occs, listener: l, counts: map[types.AssemblyJointType]int{}}
}

// Count returns the number of joints in the set.
func (s *JointSet) Count() int { return len(s.items) }

// Item returns the joint at index i (0-based), or nil when out of range.
func (s *JointSet) Item(i int) contract.AssemblyJoint {
	return collview.ItemAs(s.items, i, asContractJoint)
}

// All returns the joints in creation order.
func (s *JointSet) All() []Joint {
	out := make([]Joint, len(s.items))
	copy(out, s.items)
	return out
}

// ByID returns the joint with the given session id, or nil.
func (s *JointSet) ByID(id uint64) Joint {
	for _, j := range s.items {
		if j.ID() == id {
			return j
		}
	}
	return nil
}

// Delete removes the joint with the given id, returning whether it was found.
func (s *JointSet) Delete(id uint64) bool {
	for i, j := range s.items {
		if j.ID() != id {
			continue
		}
		s.items = append(s.items[:i], s.items[i+1:]...)
		if s.listener != nil {
			s.listener.JointDeleted(j)
		}
		return true
	}
	return false
}

// SetLimits sets (lim non-nil) or clears the joint's bounds, erroring on an unknown id.
func (s *JointSet) SetLimits(id uint64, lim *jointLimits) error {
	j := s.ByID(id)
	if j == nil {
		return fmt.Errorf("assembly: no joint with id %d", id)
	}
	j.setLimits(lim)
	return nil
}

// SetFlip sets the joint's flip sense, erroring on an unknown id.
func (s *JointSet) SetFlip(id uint64, flip bool) error {
	j := s.ByID(id)
	if j == nil {
		return fmt.Errorf("assembly: no joint with id %d", id)
	}
	j.setFlip(flip)
	return nil
}

// SetGap seats the joint's two origins the given distance apart along the joint Z-axis (#1970).
func (s *JointSet) SetGap(id uint64, gap float64) error {
	j := s.ByID(id)
	if j == nil {
		return fmt.Errorf("assembly: no joint with id %d", id)
	}
	j.setGap(gap)
	return nil
}

// SetPositions sets the joint's current linear/angular rest positions along its free DOF (#1970).
func (s *JointSet) SetPositions(id uint64, linear, angular float64) error {
	j := s.ByID(id)
	if j == nil {
		return fmt.Errorf("assembly: no joint with id %d", id)
	}
	j.setPositions(linear, angular)
	return nil
}

// SetLocked freezes (or releases) the joint's remaining free DOF (#1974).
func (s *JointSet) SetLocked(id uint64, locked bool) error {
	j := s.ByID(id)
	if j == nil {
		return fmt.Errorf("assembly: no joint with id %d", id)
	}
	j.setLocked(locked)
	return nil
}

// SetProtected marks the joint's DOF protected from other relationships (#1974).
func (s *JointSet) SetProtected(id uint64, protected bool) error {
	j := s.ByID(id)
	if j == nil {
		return fmt.Errorf("assembly: no joint with id %d", id)
	}
	j.setProtected(protected)
	return nil
}

// ForOccurrence returns the per-occurrence view of the joints referencing o.
func (s *JointSet) ForOccurrence(o *occurrence.Occurrence) *OccurrenceJoints {
	var hit []Joint
	for _, j := range s.items {
		for _, an := range j.anchors() {
			if an.occ == o {
				hit = append(hit, j)
				break
			}
		}
	}
	return &OccurrenceJoints{items: hit}
}

// relationships views the joints as the shared solver's relationships (for the combined solve).
func (s *JointSet) relationships() []relationship {
	out := make([]relationship, len(s.items))
	for i, j := range s.items {
		out[i] = j
	}
	return out
}

// add appends a joint and notifies the listener.
func (s *JointSet) add(j Joint) {
	s.items = append(s.items, j)
	if s.listener != nil {
		s.listener.JointAdded(j)
	}
}

// newJoint mints an assembly joint of the given kind over the two joint-origin refs.
func (s *JointSet) newJoint(kind types.AssemblyJointType, a, b Ref) *assemblyJoint {
	s.nextID++
	s.counts[kind]++
	return &assemblyJoint{jointBase: &jointBase{
		id: s.nextID, name: jointName(kind, s.counts[kind]), a: toAnchor(a), b: toAnchor(b),
		kind: kind,
	}}
}

// AddRigid fixes two components together (0 DOF).
func (s *JointSet) AddRigid(a, b Ref) Joint { return s.addKind(types.JointRigid, a, b) }

// AddRotational allows one rotation about the joint axis (1 DOF).
func (s *JointSet) AddRotational(a, b Ref) Joint { return s.addKind(types.JointRotational, a, b) }

// AddSlider allows one translation along the joint axis (1 DOF).
func (s *JointSet) AddSlider(a, b Ref) Joint { return s.addKind(types.JointSlider, a, b) }

// AddCylindrical allows translation along and rotation about the axis (2 DOF).
func (s *JointSet) AddCylindrical(a, b Ref) Joint { return s.addKind(types.JointCylindrical, a, b) }

// AddPlanar allows two in-plane translations and a rotation about the normal (3 DOF).
func (s *JointSet) AddPlanar(a, b Ref) Joint { return s.addKind(types.JointPlanar, a, b) }

// AddBall allows three rotations about a common point (3 DOF).
func (s *JointSet) AddBall(a, b Ref) Joint { return s.addKind(types.JointBall, a, b) }

// addKind builds, adds, and returns a joint of the given kind.
func (s *JointSet) addKind(kind types.AssemblyJointType, a, b Ref) Joint {
	j := s.newJoint(kind, a, b)
	s.add(j)
	return j
}

// OccurrenceJoints is the per-occurrence joints view (contract.AssemblyJointsEnumerator).
type OccurrenceJoints struct{ items []Joint }

// Count returns the number of joints referencing the occurrence.
func (e *OccurrenceJoints) Count() int { return len(e.items) }

// Item returns the i-th joint referencing the occurrence, or nil when out of range.
func (e *OccurrenceJoints) Item(i int) contract.AssemblyJoint {
	return collview.ItemAs(e.items, i, asContractJoint)
}

// asContractJoint widens the engine's Joint to the public contract for the shared
// collview guard (#1655).
func asContractJoint(j Joint) contract.AssemblyJoint { return j }

// jointNames maps each kind to its display prefix.
var jointNames = map[types.AssemblyJointType]string{
	types.JointRigid: "Rigid", types.JointRotational: "Rotational", types.JointSlider: "Slider",
	types.JointCylindrical: "Cylindrical", types.JointPlanar: "Planar", types.JointBall: "Ball",
}

// jointName builds a unique display name for the n-th joint of a kind.
func jointName(kind types.AssemblyJointType, n int) string {
	prefix, ok := jointNames[kind]
	if !ok {
		prefix = "Joint"
	}
	return fmt.Sprintf("%s:%d", prefix, n)
}

// SolveAssembly positions the assembly's occurrences to satisfy BOTH its constraints and its
// joints over one solve (ADR-0011), committing the result and notifying the constraint
// listener that the assembly resolved. It is the host's unified assembly solve.
func SolveAssembly(cs *ConstraintSet, js *JointSet) SolveReport {
	rep := solveOver(cs.occs, combinedRelationships(cs, js), true)
	if cs.listener != nil {
		cs.listener.AssemblyResolved()
	}
	return rep
}

// AssemblyHealth reports the assembly's combined constraint+joint health/DOF without moving.
func AssemblyHealth(cs *ConstraintSet, js *JointSet) SolveReport {
	return solveOver(cs.occs, combinedRelationships(cs, js), false)
}

// combinedRelationships gathers the active relationships from both sets.
func combinedRelationships(cs *ConstraintSet, js *JointSet) []relationship {
	rels := cs.relationships()
	if js != nil {
		rels = append(rels, js.relationships()...)
	}
	return rels
}
