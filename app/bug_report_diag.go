// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"runtime"

	"gopkg.in/yaml.v3"

	"oblikovati.org/app/options"
	"oblikovati.org/build"
	"oblikovati.org/model/doc"
	"oblikovati.org/report"
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

// TransactionLog returns the active document's undo/redo step labels, oldest first — the
// "what the user did" trail. Empty when no document is active. This is the public accessor
// over the per-document command.History the session keeps in docHistory.
func (s *Session) TransactionLog() []string {
	d := s.ActiveDocument()
	if d == nil {
		return nil
	}
	dh := s.documentHistory(d)
	if dh == nil || dh.hist == nil {
		return nil
	}
	return dh.hist.Labels()
}

// openDocumentInfos summarises every open document for the report.
func (s *Session) openDocumentInfos() []report.DocumentInfo {
	docs := s.workspace.Documents()
	out := make([]report.DocumentInfo, 0, len(docs))
	for _, d := range docs {
		out = append(out, report.DocumentInfo{
			Path:            d.FullDocumentName(),
			Name:            d.DisplayName(),
			Type:            d.DocumentType().String(),
			Dirty:           d.Dirty(),
			DisplaySettings: displaySettingsText(d),
		})
	}
	return out
}

// displaySettingsText renders a document's per-document display settings as YAML, or ""
// when the document uses the application defaults (the common case).
func displaySettingsText(d *doc.Document) string {
	set, ok := d.DisplaySettings()
	if !ok {
		return ""
	}
	b, err := yaml.Marshal(set)
	if err != nil {
		return fmt.Sprintf("(failed to render display settings: %v)", err)
	}
	return string(b)
}

// marshalOptions renders the user's typed options as YAML for the report (the same shape
// they are persisted in), so a triager sees exactly the preferences in effect.
func marshalOptions(all options.All) string {
	b, err := yaml.Marshal(all)
	if err != nil {
		return fmt.Sprintf("(failed to render options: %v)", err)
	}
	return string(b)
}
