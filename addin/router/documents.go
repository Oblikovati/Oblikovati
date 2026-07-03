// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// docInfo marshals a model document into the wire DTO, flagging whether it is the
// active one.
func docInfo(d *doc.Document, active *doc.Document) wire.DocumentInfo {
	return wire.DocumentInfo{
		ID: uint64(d.ID()), Name: d.DisplayName(), Type: d.DocumentType().String(),
		SubType: string(d.SubType()), Dirty: d.Dirty(), Visible: d.Visible(), Active: d == active,
	}
}

// listDocuments returns every open document and which one is active.
func listDocuments(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	ws := s.Workspace()
	active := ws.ActiveDocument()
	docs := ws.Documents()
	out := make([]wire.DocumentInfo, len(docs))
	for i, d := range docs {
		out[i] = docInfo(d, active)
	}
	return json.Marshal(wire.ListDocumentsResult{Documents: out})
}

// createDocument creates a new document of the given kind and makes it active. For a
// part it installs the realized component definition (as the head's seed does), so
// parameters/sketches/features are immediately usable.
func createDocument(s *app.Session, in wire.CreateDocumentArgs) (wire.DocumentInfo, error) {
	if in.Name == "" {
		return wire.DocumentInfo{}, errors.New("documents.create: name is required")
	}
	t, err := doc.ParseDocumentType(in.Type)
	if err != nil {
		return wire.DocumentInfo{}, fmt.Errorf("documents.create: %w", err)
	}
	d, err := newDocumentOfType(s, t, in.Name)
	if err != nil {
		return wire.DocumentInfo{}, err
	}
	// A requested flavor must be registered with a matching base type (M05-F15).
	if in.SubType != "" {
		if err := s.StampDocumentSubType(d, doc.SubTypeID(in.SubType)); err != nil {
			return wire.DocumentInfo{}, err
		}
		if err := enableSheetMetalIfFlavored(d, doc.SubTypeID(in.SubType)); err != nil {
			return wire.DocumentInfo{}, err
		}
	}
	return docInfo(d, d), nil
}

// enableSheetMetalIfFlavored seeds the sheet-metal rule on a part stamped with the
// sheet-metal subtype, so a part created as sheet metal enters the environment ready to
// take wall/bend features (M13-F01). Other flavors are left untouched.
func enableSheetMetalIfFlavored(d *doc.Document, sub doc.SubTypeID) error {
	if sub != types.SubTypeSheetMetalPart {
		return nil
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return fmt.Errorf("documents.create: sheet-metal subtype on a non-part document %q", d.DisplayName())
	}
	if _, err := part.EnableSheetMetal(); err != nil {
		return err
	}
	return nil
}

// registerDocumentSubType declares an add-in flavor (wire documents.registerSubType).
func registerDocumentSubType(s *app.Session, in wire.RegisterDocumentSubTypeArgs) (wire.OKResult, error) {
	base, err := app.ParseDocTypeName(in.BaseType)
	if err != nil {
		return wire.OKResult{}, err
	}
	if err := s.RegisterDocumentSubType(app.DocumentSubType{
		ID: doc.SubTypeID(in.ID), BaseType: base, DisplayName: in.DisplayName,
	}); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// listDocumentSubTypes returns the registered flavors (wire documents.listSubTypes).
func listDocumentSubTypes(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListDocumentSubTypesResult{SubTypes: s.SubTypeInfos()})
}

// activateDocument makes the identified document the active one.
func activateDocument(s *app.Session, in wire.ActivateDocumentArgs) (wire.OKResult, error) {
	for _, d := range s.Workspace().Documents() {
		if uint64(d.ID()) == in.ID {
			if err := s.Workspace().SetActiveDocument(d); err != nil {
				return wire.OKResult{}, err
			}
			return wire.OKResult{OK: true}, nil
		}
	}
	return wire.OKResult{}, fmt.Errorf("documents.activate: no open document with id %d", in.ID)
}

// closeDocument closes the identified document, discarding unsaved changes when
// force is set (otherwise it is saved first, which fails without a store).
func closeDocument(s *app.Session, in wire.CloseDocumentArgs) (wire.CloseDocumentsResult, error) {
	for _, d := range s.Workspace().Documents() {
		if uint64(d.ID()) == in.ID {
			if err := s.Workspace().Close(d, in.Force); err != nil {
				return wire.CloseDocumentsResult{}, err
			}
			return wire.CloseDocumentsResult{Closed: 1}, nil
		}
	}
	return wire.CloseDocumentsResult{}, fmt.Errorf("documents.close: no open document with id %d", in.ID)
}

// closeAllDocuments closes every open document — the way to start a clean session.
// Force discards unsaved changes. It closes a snapshot of the list so mutating the
// workspace mid-iteration is safe.
func closeAllDocuments(s *app.Session, in wire.CloseAllDocumentsArgs) (wire.CloseDocumentsResult, error) {
	closed := 0
	for _, d := range s.Workspace().Documents() {
		if err := s.Workspace().Close(d, in.Force); err != nil {
			return wire.CloseDocumentsResult{}, fmt.Errorf("documents.closeAll: closing %d: %w", d.ID(), err)
		}
		closed++
	}
	return wire.CloseDocumentsResult{Closed: closed}, nil
}

// newDocumentOfType adds a document of kind t. A part gets a realized component
// definition (compdef.AddPart); other kinds use the workspace's default content.
func newDocumentOfType(s *app.Session, t doc.DocumentType, name string) (*doc.Document, error) {
	if t == doc.Part {
		return compdef.AddPart(s.Workspace(), name, true)
	}
	return s.Workspace().Add(t, name, true)
}
