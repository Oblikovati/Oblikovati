// SPDX-License-Identifier: GPL-2.0-only

package part21

import (
	"strconv"
	"strings"
)

// QuoteString encodes s as a Part 21 single-quoted string, doubling embedded
// quotes. (Full \X\ unicode escaping is deferred; ASCII content round-trips.)
func QuoteString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// FormatReal renders f as a Part 21 real literal (always containing a '.', as the
// grammar requires for reals). It uses the shortest round-trippable form so output
// is compact yet byte-stable.
func FormatReal(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		return s + "."
	}
	// A bare exponent like "1e+20" has no '.', which Part 21 disallows for reals.
	if strings.ContainsAny(s, "eE") && !strings.Contains(s, ".") {
		return strings.Replace(s, "e", ".e", 1)
	}
	return s
}

// FormatList wraps already-formatted items as ( a , b , … ).
func FormatList(items ...string) string {
	return "(" + strings.Join(items, ",") + ")"
}

// FormatEnum wraps a token as a .TOKEN. enumeration.
func FormatEnum(token string) string { return "." + token + "." }

// FormatBool renders a Part 21 boolean enumeration.
func FormatBool(v bool) string {
	if v {
		return ".T."
	}
	return ".F."
}

// writeHeaderSection emits the HEADER section from a Header (used by Writer.Emit).
func writeHeaderSection(b *strings.Builder, h Header) {
	b.WriteString("HEADER;\n")
	b.WriteString("FILE_DESCRIPTION(" + stringListLit(h.Description) + "," + QuoteString(h.ImplementationLevel) + ");\n")
	writeFileName(b, h)
	b.WriteString("FILE_SCHEMA(" + stringListLit(h.SchemaIdentifiers) + ");\n")
	b.WriteString("ENDSEC;\n")
}

// writeFileName emits the 7-field FILE_NAME record.
func writeFileName(b *strings.Builder, h Header) {
	fields := []string{
		QuoteString(h.Name), QuoteString(h.TimeStamp),
		stringListLit(h.Author), stringListLit(h.Organization),
		QuoteString(h.PreprocessorVersion), QuoteString(h.OriginatingSystem),
		QuoteString(h.Authorization),
	}
	b.WriteString("FILE_NAME(" + strings.Join(fields, ",") + ");\n")
}

// stringListLit renders a ('a','b',…) string list literal.
func stringListLit(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = QuoteString(s)
	}
	return FormatList(quoted...)
}
