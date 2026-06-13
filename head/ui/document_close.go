// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

func closeDocumentNow(s *app.Session, d *doc.Document, skipSave bool) {
	if err := s.Workspace().Close(d, skipSave); err != nil {
		fmt.Fprintf(os.Stderr, "close %q: %v\n", d.FullDocumentName(), err)
	}
}

func documentNeedsSaveAs(d *doc.Document) bool {
	return d != nil && !doc.HasDocumentExtension(d.FullFileName())
}
