// SPDX-License-Identifier: GPL-2.0-only

// Package apidoc is the editor's signature-help and hover-doc source for the host API. Each
// scriptable wire method carries a summary and its argument fields; the editor looks them up by
// dotted name to show a parameter hint while typing a call and a tooltip when hovering a method.
//
// The data is generated at build time from the API contract (via script/luadoc, which parses the
// api module source) into data_gen.go, because that source is not present at runtime. Run
// `go generate ./script/console/apidoc` after the API surface changes.
package apidoc

//go:generate go run gen/main.go

// Param is one argument of a method: its Lua table key and a human-readable type.
type Param struct {
	Name string
	Type string
}

// Doc is the documentation for one wire method: its dotted name (as used under `oblikovati.`),
// a one-line summary, and its parameters (nil when the method takes no arguments).
type Doc struct {
	Wire    string
	Summary string
	Params  []Param
}

// Set is an immutable lookup of Docs by dotted wire name.
type Set struct {
	byWire map[string]Doc
}

// New indexes docs by wire name.
func New(docs []Doc) *Set {
	by := make(map[string]Doc, len(docs))
	for _, d := range docs {
		by[d.Wire] = d
	}
	return &Set{byWire: by}
}

// Default returns a Set over the generated doc table embedded at build time.
func Default() *Set { return New(generated) }

// Lookup returns the Doc for a dotted wire name (e.g. "sketch.rectangle") and whether it exists.
func (s *Set) Lookup(wire string) (Doc, bool) {
	d, ok := s.byWire[wire]
	return d, ok
}

// Signature renders a method's call hint, e.g. "sketch.rectangle{ width, depth }" — the params
// joined by commas inside braces (the Lua table-call form), or "method{}" when it takes none.
func (d Doc) Signature() string {
	if len(d.Params) == 0 {
		return d.Wire + "{}"
	}
	out := d.Wire + "{ "
	for i, p := range d.Params {
		if i > 0 {
			out += ", "
		}
		out += p.Name
	}
	return out + " }"
}
