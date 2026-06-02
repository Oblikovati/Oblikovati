// SPDX-License-Identifier: GPL-2.0-only

package app

// Tool is an interactive command in progress — Inventor's interactive command model
// (e.g. Extrude): activated, it collects picks/values, and on OK commits a model
// edit. Tools are pure logic driven by the session's input methods, so a test runs a
// full tool flow (pick a profile, set a distance, OK) with synthetic input and
// asserts the resulting geometry.
type Tool interface {
	// Name is the tool's display name.
	Name() string
	// Start is called when the tool is activated (e.g. to set the selection filter).
	Start(s *Session)
	// Pick receives a selectable the user clicked while the tool is active.
	Pick(s *Session, sel Selectable)
	// CanCommit reports whether enough input has been gathered to finish (enables OK).
	CanCommit() bool
	// Commit finishes the tool, applying its model edit; returns an error to keep the
	// tool open (e.g. a failed operation).
	Commit(s *Session) error
	// Cancel abandons the tool with no change.
	Cancel(s *Session)
}

// ToolInstance is a tool currently active in the session.
type ToolInstance struct {
	tool Tool
}

// Tool returns the running tool; Name its name.
func (ti *ToolInstance) Tool() Tool   { return ti.tool }
func (ti *ToolInstance) Name() string { return ti.tool.Name() }
