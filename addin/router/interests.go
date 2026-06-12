// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The add-in data registry over the wire (M03-F10, #611).

// listDocumentInterests returns a document's interest records (wire
// documents.listInterests).
func listDocumentInterests(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ListDocumentInterestsArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	d, err := documentByID(s, in.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.ListDocumentInterestsResult{Interests: d.InterestRecords()})
}

// addDocumentInterest registers (or updates) an interest record (wire
// documents.addInterest).
func addDocumentInterest(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.AddDocumentInterestArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	d, err := documentByID(s, in.Document)
	if err != nil {
		return nil, err
	}
	if err := d.Interests().Add(in.Interest); err != nil {
		return nil, err
	}
	return ok()
}

// removeDocumentInterest deletes the (clientId, name) record (wire
// documents.removeInterest).
func removeDocumentInterest(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.RemoveDocumentInterestArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	d, err := documentByID(s, in.Document)
	if err != nil {
		return nil, err
	}
	if !d.Interests().Remove(in.ClientID, in.Name) {
		return nil, fmt.Errorf("router: document %d has no interest (%q, %q)", in.Document, in.ClientID, in.Name)
	}
	return ok()
}

// hasDocumentInterest answers the discovery probe (wire documents.hasInterest).
func hasDocumentInterest(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.HasDocumentInterestArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	d, err := documentByID(s, in.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.HasDocumentInterestResult{HasInterest: d.Interests().HasInterest(in.Client)})
}
