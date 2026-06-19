//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"

	"oblikovati.org/head/internal/native"
	"oblikovati.org/script/console/apidoc"
)

// sigHelp palette and limits.
var (
	colSigBg   = [4]float32{0.16, 0.16, 0.20, 0.98}
	colSigText = [4]float32{0.85, 0.87, 0.92, 1}
	colSigDoc  = [4]float32{0.62, 0.66, 0.74, 1}
)

// maxSummaryChars caps the hovered summary so a long sentence does not run off the window.
const maxSummaryChars = 72

// signatureDoc returns the documentation for the host method whose call the caret is inside,
// resolving `oblikovati.<group>.<method>{ … }` to its wire name. ok is false when the caret is
// not inside a known host call.
func (e *codeEditor) signatureDoc() (apidoc.Doc, bool) {
	if e.apidocs == nil {
		return apidoc.Doc{}, false
	}
	chain, ok := e.model.EnclosingCall()
	if !ok || len(chain) < 2 || chain[0] != "oblikovati" {
		return apidoc.Doc{}, false
	}
	return e.apidocs.Lookup(strings.Join(chain[1:], "."))
}

// drawSignatureHelp shows the enclosing call's signature and summary in a small tooltip near the
// caret while typing arguments. It yields to the completion popup so the two never overlap.
func (e *codeEditor) drawSignatureHelp(ox, oy float32, m editorMetrics) {
	if e.completionVisible() {
		return
	}
	doc, ok := e.signatureDoc()
	if !ok {
		return
	}
	sig := doc.Signature()
	summary := truncate(doc.Summary, maxSummaryChars)
	x, y := e.sigHelpAnchor(ox, oy, m)
	w := float32(maxChars(sig, summary)+1) * m.charW
	native.DrawRectFilled(x, y, x+w, y+2*m.lineH, colSigBg)
	native.DrawTextMono(x+m.charW*0.3, y, sig, colSigText)
	native.DrawTextMono(x+m.charW*0.3, y+m.lineH, summary, colSigDoc)
}

// sigHelpAnchor places the tooltip two lines above the caret, dropping below it when there is no
// room at the top of the viewport.
func (e *codeEditor) sigHelpAnchor(ox, oy float32, m editorMetrics) (x, y float32) {
	c := e.model.Caret()
	x = e.colX(ox, 0, m)
	y = e.lineY(oy, c.Line, m) - 2*m.lineH
	if y < oy {
		y = e.lineY(oy, c.Line+1, m)
	}
	return x, y
}

// drawHoverDoc shows a method's signature and summary in a tooltip just below the pointer when
// it hovers a host method token. It yields to the completion popup.
func (e *codeEditor) drawHoverDoc(ox, oy float32, m editorMetrics, hovered bool) {
	if !hovered || e.completionVisible() || e.apidocs == nil {
		return
	}
	chain, ok := e.model.MethodChainAt(e.posAt(ox, oy, m))
	if !ok || len(chain) < 2 || chain[0] != "oblikovati" {
		return
	}
	doc, ok := e.apidocs.Lookup(strings.Join(chain[1:], "."))
	if !ok {
		return
	}
	mx, my := native.MousePos()
	sig, summary := doc.Signature(), truncate(doc.Summary, maxSummaryChars)
	w := float32(maxChars(sig, summary)+1) * m.charW
	native.DrawRectFilled(mx, my+m.lineH, mx+w, my+m.lineH+2*m.lineH, colSigBg)
	native.DrawTextMono(mx+m.charW*0.3, my+m.lineH, sig, colSigText)
	native.DrawTextMono(mx+m.charW*0.3, my+2*m.lineH, summary, colSigDoc)
}

// truncate shortens s to n runes with an ellipsis when it overflows.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// maxChars returns the longer rune length of a and b — the tooltip's width in cells.
func maxChars(a, b string) int {
	la, lb := len([]rune(a)), len([]rune(b))
	if la > lb {
		return la
	}
	return lb
}
