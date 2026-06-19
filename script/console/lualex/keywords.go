// SPDX-License-Identifier: GPL-2.0-only

package lualex

import "sort"

// luaKeywords is the reserved-word set of Lua 5.4 (the manual's §3.1). These can never be
// identifiers, so they always colour as keywords.
var luaKeywords = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true,
	"end": true, "false": true, "for": true, "function": true, "goto": true,
	"if": true, "in": true, "local": true, "nil": true, "not": true,
	"or": true, "repeat": true, "return": true, "then": true, "true": true,
	"until": true, "while": true,
}

// luaBuiltins are the standard global functions and library tables of the Lua base + standard
// libraries (manual §6), plus `oblikovati` — the single host door the Script Console exposes
// (ADR-0028). They are not reserved (a script may shadow them), so they colour as builtins
// only when not otherwise a keyword. Kept deliberately to the standard surface so the
// highlighting matches what a Lua programmer expects.
var luaBuiltins = map[string]bool{
	// base library globals
	"assert": true, "collectgarbage": true, "dofile": true, "error": true,
	"getmetatable": true, "ipairs": true, "load": true, "loadfile": true,
	"next": true, "pairs": true, "pcall": true, "print": true, "rawequal": true,
	"rawget": true, "rawlen": true, "rawset": true, "require": true, "select": true,
	"setmetatable": true, "tonumber": true, "tostring": true, "type": true,
	"xpcall": true, "_G": true, "_VERSION": true,
	// standard library tables
	"string": true, "table": true, "math": true, "os": true, "io": true,
	"coroutine": true, "utf8": true, "debug": true, "package": true,
	// the Script Console host door
	"oblikovati": true,
}

// Keywords returns the Lua reserved words in sorted order. The autocomplete engine consumes
// this so the keyword list lives in exactly one place (no duplicate table to drift).
func Keywords() []string { return sortedKeys(luaKeywords) }

// Builtins returns the known standard-library globals/tables (and the `oblikovati` door) in
// sorted order, for the autocomplete engine's bare-prefix suggestions.
func Builtins() []string { return sortedKeys(luaBuiltins) }

// sortedKeys returns the keys of set in ascending order.
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// classifyWord returns the token Kind for an identifier word: a reserved keyword, a known
// builtin global/library, or a plain identifier.
func classifyWord(word string) Kind {
	switch {
	case luaKeywords[word]:
		return KindKeyword
	case luaBuiltins[word]:
		return KindBuiltin
	default:
		return KindIdent
	}
}
