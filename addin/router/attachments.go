// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"time"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// External-file attachments over the wire (M03-F08, #609).

// attachmentInfo marshals one attachment record into the wire DTO.
func attachmentInfo(a *doc.FileAttachment) wire.AttachmentInfo {
	info := wire.AttachmentInfo{
		Name: a.Name(), Kind: a.Kind(), FullFileName: a.FullFileName(),
		ResolvedFileName: a.ResolvedFileName(), Resource: a.ResourceID(),
		Status: a.Status(), BrowserVisible: a.BrowserVisible(),
	}
	if t := a.LastKnownFileTime(); !t.IsZero() {
		info.LastKnownFileTime = t.UTC().Format(time.RFC3339Nano)
	}
	return info
}

// listAttachments returns a document's attachment records (wire
// documents.listAttachments).
func listAttachments(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.ListAttachmentsArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	d, err := documentByID(s, in.Document)
	if err != nil {
		return nil, err
	}
	all := d.Attachments().All()
	out := make([]wire.AttachmentInfo, len(all))
	for i, a := range all {
		out[i] = attachmentInfo(a)
	}
	return json.Marshal(wire.ListAttachmentsResult{Attachments: out})
}

// addAttachment attaches an external file (wire documents.addAttachment).
func addAttachment(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.AddAttachmentArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	d, err := documentByID(s, in.Document)
	if err != nil {
		return nil, err
	}
	if _, err := d.Attachments().Add(in.Name, in.Kind, in.FullFileName); err != nil {
		return nil, err
	}
	return json.Marshal(attachmentInfo(d.Attachments().Record(in.Name)))
}

// removeAttachment deletes a named attachment record (wire
// documents.removeAttachment).
func removeAttachment(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.RemoveAttachmentArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	d, err := documentByID(s, in.Document)
	if err != nil {
		return nil, err
	}
	if !d.Attachments().Remove(in.Name) {
		return nil, fmt.Errorf("router: document %d has no attachment named %q", in.Document, in.Name)
	}
	return ok()
}
