// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/doc"
)

// docInfo marshals a model document into the wire DTO, flagging whether it is the
// active one.
func docInfo(d *doc.Document, active *doc.Document) wire.DocumentInfo {
	return wire.DocumentInfo{
		ID: uint64(d.ID()), Name: d.DisplayName(), Type: d.DocumentType().String(),
		Dirty: d.Dirty(), Visible: d.Visible(), Active: d == active,
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
func createDocument(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.CreateDocumentArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, errors.New("documents.create: name is required")
	}
	t, err := parseDocumentType(in.Type)
	if err != nil {
		return nil, err
	}
	d, err := newDocumentOfType(s, t, in.Name)
	if err != nil {
		return nil, err
	}
	return json.Marshal(docInfo(d, d))
}

// activateDocument makes the identified document the active one.
func activateDocument(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ActivateDocumentArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	for _, d := range s.Workspace().Documents() {
		if uint64(d.ID()) == in.ID {
			if err := s.Workspace().SetActiveDocument(d); err != nil {
				return nil, err
			}
			return ok()
		}
	}
	return nil, fmt.Errorf("documents.activate: no open document with id %d", in.ID)
}

// newDocumentOfType adds a document of kind t. A part gets a realized component
// definition (compdef.AddPart); other kinds use the workspace's default content.
func newDocumentOfType(s *app.Session, t doc.DocumentType, name string) (*doc.Document, error) {
	if t == doc.Part {
		return compdef.AddPart(s.Workspace(), name, true)
	}
	return s.Workspace().Add(t, name, true)
}

// parseDocumentType maps a type name to the DocumentType discriminator.
func parseDocumentType(name string) (doc.DocumentType, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "part":
		return doc.Part, nil
	case "assembly":
		return doc.Assembly, nil
	case "drawing":
		return doc.Drawing, nil
	case "presentation":
		return doc.Presentation, nil
	default:
		return doc.Unknown, fmt.Errorf("documents.create: unknown type %q (want part|assembly|drawing|presentation)", name)
	}
}
