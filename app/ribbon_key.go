// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/doc"
)

// RibbonKey is the internal name of one of the seven ribbons — there is one ribbon per
// document type plus ZeroDoc, and the active ribbon is selected by the active document
// (RibbonUI_Overview). A command/control is registered against one or more ribbons; the
// shell shows exactly the ribbon for the current document.
//
// The type and its stable values are defined once in the Apache-2.0 contract
// ([types.RibbonKey], ADR-0018); this alias keeps the historical app.RibbonKey / app.PartRibbon
// spelling working unchanged across the implementation and lets add-ins target the same names.
type RibbonKey = types.RibbonKey

const (
	// ZeroDocRibbon is shown when no document is open (the Get Started ribbon).
	ZeroDocRibbon = types.ZeroDocRibbon
	// PartRibbon is shown for a part document.
	PartRibbon = types.PartRibbon
	// AssemblyRibbon is shown for an assembly document.
	AssemblyRibbon = types.AssemblyRibbon
	// DrawingRibbon is shown for a drawing document.
	DrawingRibbon = types.DrawingRibbon
	// PresentationRibbon is shown for a presentation document.
	PresentationRibbon = types.PresentationRibbon
	// IFeaturesRibbon is shown for an iFeature authoring document.
	IFeaturesRibbon = types.IFeaturesRibbon
	// UnknownDocumentRibbon is shown for a document whose type is not resolved (Inventor uses
	// it for the notebook and drawing view-orientation environments).
	UnknownDocumentRibbon = types.UnknownDocumentRibbon
)

// ribbonKeyForDocument maps the active document to the ribbon to display, falling back to
// ZeroDoc when nothing is open and UnknownDocument for an unresolved type — so the shell
// always has exactly one ribbon to render.
func ribbonKeyForDocument(d *doc.Document) RibbonKey {
	if d == nil {
		return ZeroDocRibbon
	}
	switch d.DocumentType() {
	case doc.Part:
		return PartRibbon
	case doc.Assembly:
		return AssemblyRibbon
	case doc.Drawing:
		return DrawingRibbon
	case doc.Presentation:
		return PresentationRibbon
	default:
		return UnknownDocumentRibbon
	}
}

// Environment is a ribbon context within a document. The base environment is always shown;
// a contextual environment (e.g. sketch editing) adds its tabs only while it is active — the
// mechanism behind Inventor's contextual Sketch tab (RibbonUI_Overview). Defined once in the
// Apache-2.0 contract ([types.Environment], ADR-0018) and aliased here.
type Environment = types.Environment

const (
	// BaseEnvironment is the document's normal environment; its commands always show.
	BaseEnvironment = types.BaseEnvironment
	// SketchEnvironment is active while a sketch is open for editing; its commands form the
	// contextual Sketch tab and show only then.
	SketchEnvironment = types.SketchEnvironment
)

// currentEnvironment reports the session's active ribbon environment.
func currentEnvironment(s *Session) Environment {
	if s.InSketch() {
		return SketchEnvironment
	}
	return BaseEnvironment
}

// environmentShows reports whether a command scoped to cmdEnv appears in the current
// environment: base-environment commands always show; a contextual command shows only in its
// own environment, so leaving the environment removes its tab.
func environmentShows(cmdEnv, current Environment) bool {
	return cmdEnv == BaseEnvironment || cmdEnv == current
}
