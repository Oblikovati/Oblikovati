// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// registerEndOfPartHandlers wires the part end-of-part rollback-marker methods (#141).
func (r *Router) registerEndOfPartHandlers() {
	r.handlers[wire.MethodDocumentGetEndOfPart] = getEndOfPart
	r.handlers[wire.MethodDocumentSetEndOfPart] = setEndOfPart
}

// getEndOfPart returns the active part's end-of-part marker state
// (wire.MethodDocumentGetEndOfPart).
func getEndOfPart(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(endOfPartResult(part))
}

// setEndOfPart moves the active part's end-of-part marker, re-evaluates the program up to it, and
// returns the marker's new state (wire.MethodDocumentSetEndOfPart).
func setEndOfPart(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetEndOfPartArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part.SetEndOfPart(in.Position)
	part.Recompute() // re-evaluate the feature program up to the moved marker
	return json.Marshal(endOfPartResult(part))
}

// endOfPartResult reads the part's marker into its wire reply.
func endOfPartResult(part *compdef.PartComponentDefinition) wire.EndOfPartResult {
	return wire.EndOfPartResult{Position: part.EndOfPartPosition(), RolledBack: part.IsRolledBack()}
}
