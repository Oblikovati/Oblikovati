// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/contract"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// The document open/save lifecycle over the wire (#138) and the save policy
// layer around it: SaveCopyAs and batch save (M03-F09, #610).

// openDocument loads (or returns) the document at a full document name (wire
// documents.open). DeferContent registers a reference stub without paging
// content in.
func openDocument(s *app.Session, in wire.OpenDocumentArgs) (wire.DocumentInfo, error) {
	if in.FullDocumentName == "" {
		return wire.DocumentInfo{}, fmt.Errorf("router: documents.open needs a fullDocumentName")
	}
	d, err := openDocumentWithOptions(s, in)
	if err != nil {
		return wire.DocumentInfo{}, err
	}
	return docInfo(d, s.Workspace().ActiveDocument()), nil
}

// openDocumentWithOptions picks the session's full open flow (history, view
// state, recents) for ordinary opens, the raw workspace path for stubs.
func openDocumentWithOptions(s *app.Session, in wire.OpenDocumentArgs) (*doc.Document, error) {
	if in.DeferContent {
		return s.Workspace().OpenWithOptions(in.FullDocumentName,
			doc.OpenOptions{Visible: in.Visible, DeferContent: true})
	}
	d, err := s.OpenDocument(in.FullDocumentName)
	if err != nil {
		return nil, err
	}
	d.SetVisible(in.Visible)
	return d, nil
}

// saveDocument writes a document at its current binding (wire documents.save).
func saveDocument(s *app.Session, in wire.SaveDocumentArgs) (wire.SaveDocumentResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.SaveDocumentResult{}, err
	}
	if err := s.SaveDocument(d); err != nil {
		return wire.SaveDocumentResult{}, err
	}
	return wire.SaveDocumentResult{FullDocumentName: d.FullDocumentName()}, nil
}

// saveDocumentAs writes a document under a new identity (wire documents.saveAs).
func saveDocumentAs(s *app.Session, in wire.SaveDocumentAsArgs) (wire.SaveDocumentResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.SaveDocumentResult{}, err
	}
	if err := s.SaveDocumentAs(d, in.NewFullDocumentName); err != nil {
		return wire.SaveDocumentResult{}, err
	}
	return wire.SaveDocumentResult{FullDocumentName: d.FullDocumentName()}, nil
}

// saveDocumentCopyAs writes a copy without retargeting (wire
// documents.saveCopyAs).
func saveDocumentCopyAs(s *app.Session, in wire.SaveCopyAsArgs) (wire.SaveDocumentResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.SaveDocumentResult{}, err
	}
	meta := doc.CopyMetadata{}
	if in.Metadata != nil {
		meta = doc.CopyMetadata{DisplayName: in.Metadata.DisplayName, SubType: doc.SubTypeID(in.Metadata.SubType)}
	}
	if err := s.SaveDocumentCopyAs(d, in.TargetFileName, meta); err != nil {
		return wire.SaveDocumentResult{}, err
	}
	return wire.SaveDocumentResult{FullDocumentName: in.TargetFileName}, nil
}

// batchSave queues every item and executes one operation with per-file
// outcomes (wire documents.batchSave).
func batchSave(s *app.Session, in wire.BatchSaveArgs) (wire.BatchSaveResult, error) {
	queue := s.NewBatchSave()
	for _, item := range in.Items {
		d, err := documentByID(s, item.Document)
		if err != nil {
			return wire.BatchSaveResult{}, err
		}
		if err := queue.AddFileToSave(d, item.TargetFileName); err != nil {
			return wire.BatchSaveResult{}, err
		}
	}
	outcomes, err := executeBatch(queue, in.Operation)
	if err != nil {
		return wire.BatchSaveResult{}, err
	}
	return batchResult(in.Items, outcomes), nil
}

// executeBatch dispatches the named operation.
func executeBatch(queue *app.BatchSaveQueue, operation string) ([]contract.BatchSaveOutcome, error) {
	switch operation {
	case "save":
		return queue.ExecuteSave(), nil
	case "saveAs":
		return queue.ExecuteSaveAs(), nil
	case "saveCopyAs":
		return queue.ExecuteSaveCopyAs(), nil
	default:
		return nil, fmt.Errorf("router: batch operation %q (want save|saveAs|saveCopyAs)", operation)
	}
}

// batchResult maps the per-file outcomes onto the wire shape.
func batchResult(items []wire.BatchSaveItem, outcomes []contract.BatchSaveOutcome) wire.BatchSaveResult {
	out := wire.BatchSaveResult{Results: make([]wire.BatchSaveItemResult, len(outcomes))}
	for i, o := range outcomes {
		res := wire.BatchSaveItemResult{Document: items[i].Document, FullDocumentName: o.FullDocumentName, OK: o.Err == nil}
		if o.Err != nil {
			res.Error = o.Err.Error()
		} else {
			out.Saved++
		}
		out.Results[i] = res
	}
	return out
}
