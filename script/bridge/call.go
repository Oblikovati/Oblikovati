// SPDX-License-Identifier: GPL-2.0-only

package bridge

import (
	"oblikovati/api/client"
	"oblikovati/script"
)

// CallFuncOf adapts a client.Caller into the script.CallFunc that backs
// oblikovati.call. It is the single line of glue between the API transport surface
// (client.Caller, what add-ins consume) and the script engine's host door — the
// conversion of Lua tables to/from JSON happens inside the engine (convert.go), so
// this layer only forwards the (method, JSON) pair (ADR-0028 §3).
//
// Example:
//
//	caller := bridge.NewDirectCaller(router.Handle, session)
//	globals := script.Globals{Call: bridge.CallFuncOf(caller), Methods: rtr.Methods}
func CallFuncOf(caller client.Caller) script.CallFunc {
	return func(method string, argsJSON []byte) ([]byte, error) {
		return caller.Call(method, argsJSON)
	}
}
