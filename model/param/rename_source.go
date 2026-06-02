// SPDX-License-Identifier: GPL-2.0-only

package param

import "strings"

// renameInSource rewrites occurrences of a referenced parameter name in
// expression source text, so a dependent's displayed expression tracks a
// driver's rename. It is token-aware: only identifier tokens used as references
// are replaced — unit literals after a number and function names (followed by
// '(') are left alone. On any lex error it returns src unchanged (best effort,
// since the AST reference is already renamed for evaluation).
func renameInSource(src, oldName, newName string) string {
	if oldName == newName {
		return src
	}
	tokens, err := lex(src)
	if err != nil {
		return src
	}
	var b strings.Builder
	cursor := 0
	for i, tok := range tokens {
		if !isRenamableRef(tokens, i, oldName) {
			continue
		}
		b.WriteString(src[cursor:tok.pos])
		b.WriteString(newName)
		cursor = tok.pos + len(tok.text)
	}
	b.WriteString(src[cursor:])
	return b.String()
}

// isRenamableRef reports whether token i is an identifier used as a parameter
// reference equal to name — excluding function names and unit literals.
func isRenamableRef(tokens []token, i int, name string) bool {
	tok := tokens[i]
	if tok.kind != tokIdent || tok.text != name {
		return false
	}
	if i+1 < len(tokens) && tokens[i+1].kind == tokLParen {
		return false // function call
	}
	if i > 0 && tokens[i-1].kind == tokNumber {
		if _, isUnit := lookupUnit(tok.text); isUnit {
			return false // unit literal following a number
		}
	}
	return true
}
