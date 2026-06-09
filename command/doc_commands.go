// SPDX-License-Identifier: GPL-2.0-only

package command

import "oblikovati.org/model/doc"

// Rename returns a command that sets a document's display name, capturing the
// previous name at Apply time so redo is correct after intervening edits. It is a
// concrete example of a typed model command (the richer ones — AddFeature,
// SetParameter — arrive with their milestones) and gives the undo engine real
// document state to act on.
func Rename(d *doc.Document, name string) Command {
	var previous string
	return NewFunc(
		"Rename to "+name,
		func() error {
			previous = d.DisplayName()
			d.SetDisplayName(name)
			return nil
		},
		func() error {
			d.SetDisplayName(previous)
			return nil
		},
	)
}

// SetVisibility returns a command that shows or hides a document.
func SetVisibility(d *doc.Document, visible bool) Command {
	var previous bool
	label := "Hide"
	if visible {
		label = "Show"
	}
	return NewFunc(
		label,
		func() error {
			previous = d.Visible()
			d.SetVisible(visible)
			return nil
		},
		func() error {
			d.SetVisible(previous)
			return nil
		},
	)
}
