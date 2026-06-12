// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/model/doc"
)

// Flavored document subtypes (M05-F15, #665): an add-in registers a subtype over a
// base document type; documents created with it carry the flavor (persisted in the
// manifest), and the flavor's lifecycle reaches the owner as client.operation push
// events the relay derives from the ordinary document events.

// DocumentSubType is one registered flavor.
type DocumentSubType struct {
	ID          doc.SubTypeID
	BaseType    doc.DocumentType
	DisplayName string
}

// RegisterDocumentSubType declares a flavor; re-registering the same id replaces
// its declaration (an add-in updating its display name on reload). Ids under the
// reserved built-in prefix cannot be claimed by clients (M03-F11, #612).
func (s *Session) RegisterDocumentSubType(st DocumentSubType) error {
	if st.ID == "" {
		return fmt.Errorf("app: document subtype needs an id")
	}
	if st.ID.BuiltIn() {
		return fmt.Errorf("app: subtype id %q is under the reserved %q prefix", st.ID, types.ReservedSubTypePrefix)
	}
	s.recordDocumentSubType(st)
	return nil
}

// recordDocumentSubType writes the registry entry; the built-in flavors use it
// directly to bypass the reserved-prefix guard.
func (s *Session) recordDocumentSubType(st DocumentSubType) {
	if _, exists := s.documentSubTypes[st.ID]; !exists {
		s.documentSubTypeOrder = append(s.documentSubTypeOrder, st.ID)
	}
	s.documentSubTypes[st.ID] = st
}

// registerBuiltInSubTypes seeds the host-reserved flavors at session start:
// the sheet-metal part discriminator is persisted from the first .obk files so
// it never needs retrofitting; its environment lands with M20 (M03-F11, #612).
func (s *Session) registerBuiltInSubTypes() {
	s.recordDocumentSubType(DocumentSubType{
		ID: types.SubTypeSheetMetalPart, BaseType: doc.Part, DisplayName: "Sheet Metal Part",
	})
}

// DocumentSubTypes returns the registered flavors in registration order.
func (s *Session) DocumentSubTypes() []DocumentSubType {
	out := make([]DocumentSubType, len(s.documentSubTypeOrder))
	for i, id := range s.documentSubTypeOrder {
		out[i] = s.documentSubTypes[id]
	}
	return out
}

// StampDocumentSubType marks a document with a REGISTERED flavor whose base type
// matches — the documents.create path calls it for a requested subType.
func (s *Session) StampDocumentSubType(d *doc.Document, subType doc.SubTypeID) error {
	st, ok := s.documentSubTypes[subType]
	if !ok {
		return fmt.Errorf("app: document subtype %q is not registered", subType)
	}
	if st.BaseType != d.DocumentType() {
		return fmt.Errorf("app: subtype %q is based on %v, not %v", subType, st.BaseType, d.DocumentType())
	}
	d.SetSubType(subType)
	return nil
}

// SubTypeInfos maps the registry to the wire shape.
func (s *Session) SubTypeInfos() []wire.DocumentSubTypeInfo {
	flavors := s.DocumentSubTypes()
	out := make([]wire.DocumentSubTypeInfo, len(flavors))
	for i, st := range flavors {
		out[i] = wire.DocumentSubTypeInfo{
			ID: string(st.ID), BaseType: docTypeName(st.BaseType), DisplayName: st.DisplayName,
		}
	}
	return out
}

// docTypeName renders a document type as its wire name.
func docTypeName(t doc.DocumentType) string {
	switch t {
	case doc.Part:
		return "part"
	case doc.Assembly:
		return "assembly"
	case doc.Drawing:
		return "drawing"
	case doc.Presentation:
		return "presentation"
	default:
		return fmt.Sprintf("documentType(%d)", t)
	}
}

// ParseDocTypeName resolves a wire document-type name.
func ParseDocTypeName(name string) (doc.DocumentType, error) {
	switch name {
	case "part":
		return doc.Part, nil
	case "assembly":
		return doc.Assembly, nil
	case "drawing":
		return doc.Drawing, nil
	case "presentation":
		return doc.Presentation, nil
	default:
		return 0, fmt.Errorf("app: unknown document type %q (part/assembly/drawing/presentation)", name)
	}
}
