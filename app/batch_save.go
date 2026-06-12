// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/model/doc"
)

// BatchSaveQueue is the FileSaveAs-style batch service (M03-F09, #610): queue
// (document → target) pairs, then execute ONE operation over all of them with
// per-file outcomes — execution continues past individual failures so one bad
// path does not abort an export of fifty files. A queue is single-use:
// executing drains it.
type BatchSaveQueue struct {
	s     *Session
	items []batchSaveItem
}

type batchSaveItem struct {
	d      *doc.Document
	target string
}

var _ contract.BatchSave = (*BatchSaveQueue)(nil)

// NewBatchSave returns an empty batch-save queue.
func (s *Session) NewBatchSave() *BatchSaveQueue { return &BatchSaveQueue{s: s} }

// AddFileToSave queues a document with its target file name (ignored by
// ExecuteSave). Duplicate targets are rejected — two writes to one path is
// always a caller bug.
func (q *BatchSaveQueue) AddFileToSave(document contract.Document, targetFileName string) error {
	d, ok := document.(*doc.Document)
	if !ok || d == nil {
		return fmt.Errorf("app: batch save needs an open workspace document, got %T", document)
	}
	for _, item := range q.items {
		if item.target != "" && item.target == targetFileName {
			return fmt.Errorf("app: batch already saves to %q", targetFileName)
		}
	}
	q.items = append(q.items, batchSaveItem{d: d, target: targetFileName})
	return nil
}

// Count returns the number of queued pairs.
func (q *BatchSaveQueue) Count() int { return len(q.items) }

// ExecuteSave saves every queued document at its current binding.
func (q *BatchSaveQueue) ExecuteSave() []contract.BatchSaveOutcome {
	return q.execute(func(item batchSaveItem) (string, error) {
		return item.d.FullFileName(), q.s.SaveDocument(item.d)
	})
}

// ExecuteSaveAs saves every queued document under its target name,
// retargeting each document's identity.
func (q *BatchSaveQueue) ExecuteSaveAs() []contract.BatchSaveOutcome {
	return q.execute(func(item batchSaveItem) (string, error) {
		return item.target, q.s.SaveDocumentAs(item.d, item.target)
	})
}

// ExecuteSaveCopyAs writes a copy of every queued document to its target
// without retargeting the in-memory documents.
func (q *BatchSaveQueue) ExecuteSaveCopyAs() []contract.BatchSaveOutcome {
	return q.execute(func(item batchSaveItem) (string, error) {
		return item.target, q.s.SaveDocumentCopyAs(item.d, item.target, doc.CopyMetadata{})
	})
}

// execute drains the queue through one per-item operation.
func (q *BatchSaveQueue) execute(op func(batchSaveItem) (string, error)) []contract.BatchSaveOutcome {
	out := make([]contract.BatchSaveOutcome, len(q.items))
	for i, item := range q.items {
		name, err := op(item)
		out[i] = contract.BatchSaveOutcome{FullDocumentName: name, Err: err}
	}
	q.items = nil
	return out
}
