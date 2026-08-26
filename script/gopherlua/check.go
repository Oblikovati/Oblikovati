// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import (
	"strings"

	"github.com/yuin/gopher-lua/parse"

	"oblikovati.org/script/console/diag"
)

// SyntaxChecker is a diag.Checker that runs gopher-lua's parser over a source and reports any
// syntax error — without compiling to bytecode or executing, so it is safe to run on every edit
// pause. It lives here because this package owns the gopher-lua dependency (the editor's diag
// package stays parser-agnostic). A clean parse yields no diagnostics.
type SyntaxChecker struct{}

// Check parses source and returns at most one diagnostic (the parser stops at the first error),
// mapping the parser's 1-based line and EOF sentinel onto the editor's 0-based, clamped lines.
func (SyntaxChecker) Check(source string) []diag.Diagnostic {
	if _, err := parse.Parse(strings.NewReader(source), "console"); err != nil {
		return []diag.Diagnostic{toDiagnostic(err, source)}
	}
	return nil
}

// toDiagnostic converts a parser error into an editor diagnostic, extracting the position from a
// *parse.Error and falling back to the document start for any other error shape.
func toDiagnostic(err error, source string) diag.Diagnostic {
	pe, ok := err.(*parse.Error)
	if !ok {
		return diag.Diagnostic{Line: 0, Col: 0, Message: err.Error()}
	}
	line := clampLine(pe.Pos.Line-1, source)
	col := max(pe.Pos.Column, 0)
	return diag.Diagnostic{Line: line, Col: col, Message: pe.Message}
}

// clampLine maps a (possibly EOF-sentinel or 1-past-end) line onto a real 0-based line index in
// source, so an "unexpected EOF" error underlines the last line rather than nowhere.
func clampLine(line int, source string) int {
	last := strings.Count(source, "\n")
	if line < 0 || line > last {
		return last
	}
	return line
}
