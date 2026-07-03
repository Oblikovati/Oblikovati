// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The add-in data registry over the wire (M03-F10, #611).

// listDocumentInterests returns a document's interest records (wire
// documents.listInterests).
func listDocumentInterests(s *app.Session, in wire.ListDocumentInterestsArgs) (wire.ListDocumentInterestsResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.ListDocumentInterestsResult{}, err
	}
	return wire.ListDocumentInterestsResult{Interests: d.InterestRecords()}, nil
}

// addDocumentInterest registers (or updates) an interest record (wire
// documents.addInterest).
func addDocumentInterest(s *app.Session, in wire.AddDocumentInterestArgs) (wire.OKResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.OKResult{}, err
	}
	if err := d.Interests().Add(in.Interest); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// removeDocumentInterest deletes the (clientId, name) record (wire
// documents.removeInterest).
func removeDocumentInterest(s *app.Session, in wire.RemoveDocumentInterestArgs) (wire.OKResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.OKResult{}, err
	}
	if !d.Interests().Remove(in.ClientID, in.Name) {
		return wire.OKResult{}, fmt.Errorf("router: document %d has no interest (%q, %q)", in.Document, in.ClientID, in.Name)
	}
	return wire.OKResult{OK: true}, nil
}

// hasDocumentInterest answers the discovery probe (wire documents.hasInterest).
func hasDocumentInterest(s *app.Session, in wire.HasDocumentInterestArgs) (wire.HasDocumentInterestResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.HasDocumentInterestResult{}, err
	}
	return wire.HasDocumentInterestResult{HasInterest: d.Interests().HasInterest(in.Client)}, nil
}
