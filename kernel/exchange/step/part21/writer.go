// SPDX-License-Identifier: GPL-2.0-only

package part21

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Writer accumulates DATA-section entities and emits a byte-stable Part 21 file. It
// allocates monotonic ids and dedups identical statements (the geommap layer uses
// this to share CARTESIAN_POINTs/DIRECTIONs). Output is deterministic for a given
// emission order, which the round-trip test relies on.
//
// Example:
//
//	w := part21.NewWriter()
//	p := w.Add("CARTESIAN_POINT", "''", "(0.,0.,0.)")
//	bytes := w.Emit(hdr)
type Writer struct {
	next  int
	stmts []statement
	dedup map[string]int // statement text → id, for point/direction sharing
}

// statement is one queued #id = KEYWORD(params); record.
type statement struct {
	id   int
	body string // "KEYWORD(p1,p2,…)"
}

// NewWriter starts a writer with id allocation beginning at 1.
func NewWriter() *Writer {
	return &Writer{next: 1, dedup: map[string]int{}}
}

// Ref formats an id as a #reference, the token other entities embed.
func Ref(id int) string { return "#" + strconv.Itoa(id) }

// Add appends a statement KEYWORD(params...) and returns its freshly-allocated id.
// Params are already-formatted Part 21 tokens (use the format helpers).
func (w *Writer) Add(keyword string, params ...string) int {
	body := keyword + "(" + strings.Join(params, ",") + ")"
	id := w.next
	w.next++
	w.stmts = append(w.stmts, statement{id: id, body: body})
	return id
}

// AddRaw appends a pre-formatted statement body verbatim (e.g. a complex instance
// "(A()B()C())" that is not a single KEYWORD(params)), returning its id.
func (w *Writer) AddRaw(body string) int {
	id := w.next
	w.next++
	w.stmts = append(w.stmts, statement{id: id, body: body})
	return id
}

// AddShared appends a statement only if an identical one was not already added,
// returning the existing id on a hit. Used to share leaf geometry (points/dirs).
func (w *Writer) AddShared(keyword string, params ...string) int {
	body := keyword + "(" + strings.Join(params, ",") + ")"
	if id, ok := w.dedup[body]; ok {
		return id
	}
	id := w.next
	w.next++
	w.stmts = append(w.stmts, statement{id: id, body: body})
	w.dedup[body] = id
	return id
}

// Emit renders the full file: ISO header, HEADER section (from h), DATA section
// (statements in id order), and the trailer.
func (w *Writer) Emit(h Header) []byte {
	var b strings.Builder
	b.WriteString("ISO-10303-21;\n")
	writeHeaderSection(&b, h)
	b.WriteString("DATA;\n")
	w.writeStatements(&b)
	b.WriteString("ENDSEC;\n")
	b.WriteString("END-ISO-10303-21;\n")
	return []byte(b.String())
}

// writeStatements writes every queued statement sorted by id (stable output).
func (w *Writer) writeStatements(b *strings.Builder) {
	stmts := append([]statement(nil), w.stmts...)
	sort.Slice(stmts, func(i, j int) bool { return stmts[i].id < stmts[j].id })
	for _, s := range stmts {
		fmt.Fprintf(b, "#%d=%s;\n", s.id, s.body)
	}
}
