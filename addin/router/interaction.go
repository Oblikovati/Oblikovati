// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati/api/wire"
	"oblikovati/app"
)

// interactionState reports whether the local user is mid-action, so a collaboration
// add-in can gate/buffer incoming remote edits (wire.MethodInteractionState).
//
// Busy is true when an interactive tool is running or a bounded transaction is open.
// ActiveCommand is intentionally left empty for now: the router runs on the session
// goroutine, so a command executed via commands.execute completes before any concurrent
// query could observe it — the durable interactive state is the active tool.
func interactionState(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	st := wire.InteractionState{InTransaction: s.InTransaction()}
	if t := s.ActiveTool(); t != nil {
		st.ActiveTool = t.Name()
	}
	st.Busy = st.ActiveTool != "" || st.InTransaction
	return json.Marshal(st)
}
