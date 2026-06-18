// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// tidParameterChanged is the session-bus type id of [ParameterChanged] (0x07xx = M07 parameters).
const tidParameterChanged event.TypeID = 0x0701

// ParameterChanged announces that a parameter's expression/value changed (#148) — the granular
// notification beyond the generic edit. The fields carry the parameter's new state so a relay can
// forward it to an add-in without re-querying.
type ParameterChanged struct {
	Document   doc.ID
	Name       string
	Kind       string
	Expression string
	Value      string
}

// EventID implements event.Event.
func (ParameterChanged) EventID() event.TypeID { return tidParameterChanged }
