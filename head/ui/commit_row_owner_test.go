// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"strings"
	"testing"
)

// M40 audit S7 (#1642): drawCommitCancelButtons is the single owner of the tool-commit OK/Cancel
// row, so the sick-config gate (disable OK + amber reason) can never drift across hand-rolled
// copies. This structural guard fails when any head/ui file draws its own tool-commit OK button
// instead of routing through the shared row — the drift the audit found in the feature-edit dialog,
// command window and sheet canvas. It reads the package source as text, so it runs without the cgo
// native layer.

// commitRowOwner is the only file allowed to draw the shared tool-commit OK button.
const commitRowOwner = "dialog_buttons.go"

// nonCommitOKButtons are the files whose native.Button("OK") is a modal value editor's confirm — a
// parameter/tolerance list or an inline dimension entry — NOT a tool commit, so the sick-config gate
// does not apply and they legitimately keep their own OK. Any NEW file with a raw OK button must
// either route through drawCommitCancelButtons or be justified here on purpose.
var nonCommitOKButtons = map[string]string{
	"parameters_dialogs.go": "value-list / tolerance modal editors (applyValueList/applyTolerance, not s.OK())",
	"dimension_overlay.go":  "inline dimension value entry (CommitPendingDimension, not s.OK())",
}

func TestOKButtonHasOneOwner(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		assertNoStrayOKButton(t, name)
	}
}

// assertNoStrayOKButton flags a raw native.Button("OK") outside the shared row's owner and the
// justified modal-editor allow-list.
func assertNoStrayOKButton(t *testing.T, name string) {
	t.Helper()
	if name == commitRowOwner {
		return
	}
	src, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if !strings.Contains(string(src), `native.Button("OK")`) {
		return
	}
	if _, allowed := nonCommitOKButtons[name]; allowed {
		return
	}
	t.Errorf("%s draws its own native.Button(\"OK\"): route the tool-commit row through "+
		"drawCommitCancelButtons (the sick-config gate owner) or justify it in nonCommitOKButtons (#1642)", name)
}
