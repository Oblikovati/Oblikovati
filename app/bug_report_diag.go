// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"runtime"

	"oblikovati.org/app/options"
	"oblikovati.org/build"
	"oblikovati.org/model/doc"
	"oblikovati.org/report"
	"oblikovati.org/yamlcodec"
)

// CollectDiagnostics gathers everything a triager needs to reproduce a bug — the user's
// options, the open documents, the platform and build, and the active document's
// transaction history — into a report.Payload. The two screenshots are filled in later by
// the capture step; the comment by BeginBugReport.
func (s *Session) CollectDiagnostics() report.Payload {
	return report.Payload{
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		AppVersion:     build.Version,
		AppCommit:      build.Commit,
		AppBuildDate:   build.Date,
		UserSettings:   marshalOptions(s.appOptions),
		OpenDocuments:  s.openDocumentInfos(),
		TransactionLog: s.TransactionLog(),
	}
}

// TransactionLog returns the transaction-manager events recorded since the application
// opened — every committed edit across all documents, oldest first — each carrying the
// document's full recipe (the complete, replayable command payload). It is an append-only
// audit (see watchTransactions), not a document's current undo stack: undo does not erase
// entries here, so the report shows the full sequence of what the user did.
func (s *Session) TransactionLog() []report.TransactionEvent {
	out := make([]report.TransactionEvent, 0, len(s.txEvents))
	for _, e := range s.txEvents {
		out = append(out, report.TransactionEvent{
			Time:     e.when.Format("15:04:05"),
			Document: e.doc,
			Label:    e.label,
			Recipe:   s.auditRecipe(e),
		})
	}
	return out
}

// auditRecipe reconstructs one audit event's after-step recipe from its document's delta log
// (#1424), or "" when the step recorded no recipe (non-recipe content, a marshal failure, or a
// position since trimmed by the audit bound). The bug report renders the reconstructed recipe
// exactly as the old full-copy form did, so a triager still sees the replayable command payload.
func (s *Session) auditRecipe(e sessionTxEvent) string {
	if e.pos < 0 {
		return ""
	}
	log, ok := s.txAudit[e.docID]
	if !ok {
		return ""
	}
	r, err := log.At(e.pos)
	if err != nil {
		return ""
	}
	return string(r)
}

// documentMarshaler is the optional capability of the session's store that renders a
// document to its .obk YAML in memory (persistence.PackageStore implements it). Declared
// here so the app stays decoupled from the persistence package (the DI rule); an in-memory
// session whose store lacks it simply ships documents without their YAML content.
type documentMarshaler interface {
	MarshalDocument(d *doc.Document) ([]byte, error)
}

// openDocumentInfos summarises every open document for the report, the ACTIVE document
// first, each carrying its full .obk YAML in Content when the store can render it.
func (s *Session) openDocumentInfos() []report.DocumentInfo {
	marshaler, _ := s.store.(documentMarshaler) // nil for stores that cannot serialize (e.g. in-memory)
	active := s.ActiveDocument()
	docs := s.openDocumentsActiveFirst(active)
	out := make([]report.DocumentInfo, 0, len(docs))
	for _, d := range docs {
		out = append(out, report.DocumentInfo{
			Path:    d.FullDocumentName(),
			Name:    d.DisplayName(),
			Type:    d.DocumentType().String(),
			Dirty:   d.Dirty(),
			Active:  active != nil && d.ID() == active.ID(),
			Content: documentYAML(marshaler, d),
		})
	}
	return out
}

// openDocumentsActiveFirst returns the open documents with active first (the order the
// report renders them), or workspace order when there is no active document.
func (s *Session) openDocumentsActiveFirst(active *doc.Document) []*doc.Document {
	docs := s.workspace.Documents()
	if active == nil {
		return docs
	}
	ordered := make([]*doc.Document, 0, len(docs))
	ordered = append(ordered, active)
	for _, d := range docs {
		if d.ID() != active.ID() {
			ordered = append(ordered, d)
		}
	}
	return ordered
}

// documentYAML renders one document's .obk content via the store, or "" when no marshaler
// is available (an in-memory session). A marshal error is surfaced inline so the report
// still carries the rest of the document set.
func documentYAML(m documentMarshaler, d *doc.Document) string {
	if m == nil {
		return ""
	}
	b, err := m.MarshalDocument(d)
	if err != nil {
		return fmt.Sprintf("(failed to serialize document: %v)", err)
	}
	return string(b)
}

// marshalOptions renders the user's typed options as YAML for the report (the same shape
// they are persisted in), so a triager sees exactly the preferences in effect.
func marshalOptions(all options.All) string {
	b, err := yamlcodec.Marshal(all)
	if err != nil {
		return fmt.Sprintf("(failed to render options: %v)", err)
	}
	return string(b)
}
