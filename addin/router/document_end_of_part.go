// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// registerEndOfPartHandlers wires the part end-of-part rollback-marker methods (#141).
func (r *Router) registerEndOfPartHandlers() {
	r.readOnly(wire.MethodDocumentGetEndOfPart, partQuery(getEndOfPart))
	r.mutating(wire.MethodDocumentSetEndOfPart, "Set End of Part", typedPart(setEndOfPart))
}

// getEndOfPart returns the active part's end-of-part marker state
// (wire.MethodDocumentGetEndOfPart).
func getEndOfPart(_ *app.Session, part *compdef.PartComponentDefinition) (wire.EndOfPartResult, error) {
	return endOfPartResult(part), nil
}

// setEndOfPart moves the active part's end-of-part marker, re-evaluates the program up to it, and
// returns the marker's new state (wire.MethodDocumentSetEndOfPart).
func setEndOfPart(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetEndOfPartArgs) (wire.EndOfPartResult, error) {
	part.SetEndOfPart(in.Position)
	part.Recompute() // re-evaluate the feature program up to the moved marker
	return endOfPartResult(part), nil
}

// endOfPartResult reads the part's marker into its wire reply.
func endOfPartResult(part *compdef.PartComponentDefinition) wire.EndOfPartResult {
	return wire.EndOfPartResult{Position: part.EndOfPartPosition(), RolledBack: part.IsRolledBack()}
}
