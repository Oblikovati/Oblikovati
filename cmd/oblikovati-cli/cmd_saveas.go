// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"io"

	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/persistence"
)

// cmdSaveAs opens src and re-saves it under dst, exercising the open→save-as
// round-trip (the rename/copy path) end to end:
//
//	oblikovati-cli save-as fixtures/bracket.obk fixtures/copy.obk
func cmdSaveAs(args []string, out io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("save-as: expected <src> <dst>, got %d arg(s)", len(args))
	}
	src, dst := args[0], withPackageExt(args[1])
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	d, err := ws.Open(src, true)
	if err != nil {
		return fmt.Errorf("save-as: open %q: %w", src, err)
	}
	if err := ws.SaveAs(d, dst); err != nil {
		return fmt.Errorf("save-as: %w", err)
	}
	fmt.Fprintf(out, "saved %q as %s\n", d.DisplayName(), dst)
	return nil
}
