// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// Where a new view's projection comes from.
//
// Application Options ▸ Display has carried a NewWindowProjection setting since M16 — persisted,
// round-tripped through the API, and already SHIPPING orthographic as its default — but nothing
// ever read it when a view was made, so every view fell to doc.ProjectionMode's zero value and the
// viewport was always perspective. CAD modelling wants a parallel projection: perspective
// foreshortening makes equal features look unequal and puts the apparent silhouette of a cylinder
// off its true one, so a dimension you read off the screen is not the dimension you drew.
//
// This connects the setting to the views the session creates, so the option is the single source of
// the default and a user who prefers perspective gets it.

// watchNewDocumentProjection applies the display option's projection to a document's initial view
// as soon as the document is created, so a brand-new part opens in the projection the user chose.
func (s *Session) watchNewDocumentProjection() {
	event.Subscribe(s.workspace.Events(), event.After, func(_ event.Context, e doc.DocumentCreated) event.Outcome {
		s.applyNewWindowProjection(e.Document)
		return event.Continue()
	})
}

// applyNewWindowProjection sets every view of a freshly created document to the configured
// new-window projection. Only creation goes through here: an OPENED document restores whatever
// projection it was saved with (loadViewState), which is the user's per-document choice and must
// outrank the global default.
func (s *Session) applyNewWindowProjection(d *doc.Document) {
	if d == nil {
		return
	}
	mode := s.newViewProjection()
	for _, v := range d.Views().All() {
		v.Projection = mode
	}
}

// newViewProjection is the projection mode a view the session creates starts in, translated from
// the Application Options ▸ Display setting.
//
//	s.newViewProjection() // doc.ProjOrthographic with the shipped default
func (s *Session) newViewProjection() doc.ProjectionMode {
	switch s.displayOptions.NewWindowProjection {
	case types.PerspectiveProjection:
		return doc.ProjPerspective
	case types.PerspectiveWithOrthoFacesProjection:
		return doc.ProjPerspectiveOrthoFaces
	default:
		return doc.ProjOrthographic
	}
}
