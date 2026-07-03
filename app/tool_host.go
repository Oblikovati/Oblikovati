// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/compdef"

// ToolHost is the slim, consumer-side seam a part-feature tool drives to commit its
// edit — the narrow subset of the session a tool legitimately needs, instead of the
// whole *Session (audit I12, #1635). *Session satisfies it implicitly (asserted below),
// so a tool's commit logic depends on this contract, not on everything a session can do,
// and a fake host lets a tool's commit run under test without constructing the whole app.
//
// It is deliberately app-internal (an unexported method seals it to this package — this
// is a tool↔session seam, not the public add-in contract) and grown only by evidence:
// the current members are exactly what the converted create-commit paths of Extrude,
// Fillet and Chamfer use. New members are added when a converting tool proves the need,
// and TestToolHostStaysSlim pins the ceiling so it cannot silently re-fatten into a
// second Session.
//
// Return-position caveat (noted, not solved here per the I12 plan): ActivePart returns
// the concrete *compdef.PartComponentDefinition, so a fake host still hands back a real
// part. Narrowing that return type is a later step, driven by whichever converted tool's
// fake first finds the concrete part painful to build.
//
// Example (a tool's commit, host-driven):
//
//	part, err := host.ActivePart()
//	if err != nil { return err }
//	f := feature.NewExtrudeFeatures(part.Features()).AddExtrudeFeature(def)
//	part.Recompute()
//	host.recordEdit(part, "Extrude")
type ToolHost interface {
	// ActivePart returns the active document's part component definition, erroring when
	// there is no active document or it is not a part.
	ActivePart() (*compdef.PartComponentDefinition, error)
	// recordEdit records the part's recipe delta as one undo step under the given label.
	recordEdit(content recipeStore, label string)
}

// *Session is the production ToolHost; a tool never sees more of the session than this.
var _ ToolHost = (*Session)(nil)

// ActivePart exposes the active part component definition to a ToolHost consumer — the
// exported face of the package-level activePart helper the tools already used (#1635).
func (s *Session) ActivePart() (*compdef.PartComponentDefinition, error) {
	return activePart(s)
}

// hostedTool is a part-feature tool whose commit has been migrated to the ToolHost seam:
// its create-commit runs against ToolHost, not *Session. The plain Tool.Commit remains
// for the input plumbing and the edit-mode path (still session-bound); this capability
// marks the converted create path and drives the I12 ratchet (TestHostedToolRatchet).
type hostedTool interface {
	// CommitFeature commits the tool's new feature through the host, returning an error
	// (a sick feature) to keep the tool open.
	CommitFeature(h ToolHost) error
}
