// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/event"
	"oblikovati.org/model/assembly"
)

// Assembly joint event type ids (M12-F02). The high byte 0x0C mirrors the milestone (M12);
// the 0x0C1x band is the joint family, after the 0x0C0x constraint family. Stable across
// versions (they cross the seam to add-ins). They ride the assembly's occurrence bus.
const (
	tidJointAdd    event.TypeID = 0x0C11
	tidJointDelete event.TypeID = 0x0C12
)

// JointAdd is raised (After) when a joint is added to the assembly.
type JointAdd struct{ Joint contract.AssemblyJoint }

// EventID implements event.Event.
func (JointAdd) EventID() event.TypeID { return tidJointAdd }

// JointDelete is raised (After) when a joint is removed from the assembly.
type JointDelete struct{ Joint contract.AssemblyJoint }

// EventID implements event.Event.
func (JointDelete) EventID() event.TypeID { return tidJointDelete }

// AssemblyEvents is the sink the joint set notifies (M12-F02).
var _ assembly.JointListener = (*AssemblyEvents)(nil)

// JointAdded raises JointAdd (After). Implements assembly.JointListener.
func (e *AssemblyEvents) JointAdded(j contract.AssemblyJoint) {
	event.Emit(e.bus, event.After, JointAdd{Joint: j})
}

// JointDeleted raises JointDelete (After).
func (e *AssemblyEvents) JointDeleted(j contract.AssemblyJoint) {
	event.Emit(e.bus, event.After, JointDelete{Joint: j})
}
