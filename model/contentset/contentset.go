// SPDX-License-Identifier: GPL-2.0-only

// Package contentset assembles the default document-content factory set — the
// composition-root wiring that pairs each document kind with its real content
// constructor (#1617, audit B6). It exists as its own package because doc cannot
// import the content packages (cycle) and because SEVERAL roots compose the model
// directly (app.NewSession, the oblikovati-cli commands, benchgen): each passes
// Default() to doc.NewWorkspace instead of relying on init()-time registration,
// so which kinds open live is decided by explicit construction, not by a binary's
// import list.
package contentset

import (
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/drawing"
)

// Default returns the full production content-factory set: parts and assemblies
// open as live component definitions, drawings as live sheet content.
//
//	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
func Default() doc.ContentFactories {
	return doc.ContentFactories{
		doc.Part:     func() doc.Content { return compdef.NewPartComponentDefinition() },
		doc.Assembly: func() doc.Content { return compdef.NewAssemblyComponentDefinition() },
		doc.Drawing:  func() doc.Content { return drawing.NewContent() },
	}
}
