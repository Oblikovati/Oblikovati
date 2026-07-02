// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"io"

	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// cmdSaveAs opens src and re-saves it under dst, exercising the open→save-as
// round-trip (the rename/copy path) end to end:
//
//	oblikovati-cli save-as fixtures/bracket.opd fixtures/copy.opd
func cmdSaveAs(args []string, out io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("save-as: expected <src> <dst>, got %d arg(s)", len(args))
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	d, err := ws.Open(args[0], true)
	if err != nil {
		return fmt.Errorf("save-as: open %q: %w", args[0], err)
	}
	dst := withDocExt(args[1], d.DocumentType()) // the copy keeps the source's kind, so its extension too
	if err := ws.SaveAs(d, dst); err != nil {
		return fmt.Errorf("save-as: %w", err)
	}
	fmt.Fprintf(out, "saved %q as %s\n", d.DisplayName(), dst)
	return nil
}
