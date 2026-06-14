// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/app/cmdline"

// CommandTokenKind tags which field of a [CommandToken] carries the parsed input.
type CommandTokenKind int

const (
	// CoordToken carries a point in Coord (always absolute — the engine resolves
	// AutoCAD "@relative" input against the previous point before feeding the tool).
	CoordToken CommandTokenKind = iota
	// ValueToken carries a bare number in Value (a distance, radius, depth, or angle).
	ValueToken
	// KeywordToken carries a chosen bracketed option in Keyword (e.g. "Close").
	KeywordToken
)

// CommandToken is one parsed command-line input the engine feeds to a [CommandDriven]
// tool. Exactly one field is meaningful, per Kind.
type CommandToken struct {
	Kind    CommandTokenKind
	Coord   cmdline.Coord
	Value   float64
	Keyword string
}

// CommandDriven is an optional [Tool] capability (M26): a tool that can take typed input
// from the Command Window implements it by accepting one parsed token per step. The same
// step the viewport drives by clicking, the command line drives by typing, so the two
// co-drive one tool (AutoCAD's "type or pick"). The step prompt comes from the tool's
// existing [Prompted] implementation, so making a tool command-drivable is just this one
// method. A tool that doesn't implement it still starts from a typed command word; its
// steps are then completed by picking in the viewport.
//
//	// drive a line entirely from text:
//	cl.Submit(s, "LINE"); cl.Submit(s, "0,0"); cl.Submit(s, "10,0")
type CommandDriven interface {
	// SubmitToken applies one parsed token and advances the tool, returning an error
	// (kept visible on the command line) when the token does not fit the step.
	SubmitToken(s *Session, tok CommandToken) error
}

// commandOptioned is the optional companion capability: a command-driven tool whose
// current step offers bracketed keyword options (e.g. [Close/Undo]). Tools that only take
// a coordinate or value omit it.
type commandOptioned interface {
	CommandOptions(s *Session) []string
}
