// SPDX-License-Identifier: GPL-2.0-only

// Package luadoc generates the Lua Scripting wiki manual from the public API: it reuses the
// exact wire surface the Lua bridge exposes (oblikovati.<group>.<method>{…}) and the
// mcp:tool / mcp:summary annotations the API already carries for the MCP bridge, so the
// scripting reference is generated from one source of truth and never drifts (ADR-0028).
//
// The Lua binding reaches every wire method, so the reference is the wire method set,
// enriched with each method's summary (from api/client) and its argument fields (from the
// api/wire request DTO). The page also embeds a hand-written guide and the runnable example
// library.
package luadoc

import (
	"fmt"
	"sort"
	"strings"
)

// Field is one argument of a method's request DTO: the Lua table key (the JSON field name)
// and a human-readable Go type.
type Field struct {
	Name string
	Type string
}

// Method is one scriptable wire method, as the manual presents it.
type Method struct {
	Wire    string  // dotted wire name, e.g. "documents.create"
	Group   string  // segment before the first dot, e.g. "documents"
	Leaf    string  // remainder, e.g. "create"
	Summary string  // mcp:summary text ("" when the API carries none)
	Args    []Field // request DTO fields (nil when the method takes no args)
}

// Example is one bundled Lua program shown in the manual.
type Example struct {
	Name        string // filename, e.g. "extrude_block.lua"
	Description string // leading-comment summary
	Source      string // full Lua source
}

// Generate renders the whole Lua Scripting manual as Markdown. apiDir is the on-disk root of
// the oblikovati.org/api module (resolve it with `go list -m -f {{.Dir}} oblikovati.org/api`).
// examples are the bundled programs to embed (script/examples).
func Generate(apiDir string, examples []Example) (string, error) {
	methods, err := collectMethods(apiDir)
	if err != nil {
		return "", fmt.Errorf("luadoc: %w", err)
	}
	if len(methods) == 0 {
		return "", fmt.Errorf("luadoc: parsed zero methods from %s/client — wrong api dir?", apiDir)
	}
	var b strings.Builder
	b.WriteString(guide())
	writeExamples(&b, examples)
	writeReference(&b, methods)
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

// Methods returns the scriptable methods parsed from the API, sorted by wire name — exposed
// for tests that assert coverage.
func Methods(apiDir string) ([]Method, error) {
	ms, err := collectMethods(apiDir)
	if err != nil {
		return nil, err
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].Wire < ms[j].Wire })
	return ms, nil
}

// groupMethods buckets methods by their wire group, each bucket sorted by leaf name.
func groupMethods(methods []Method) ([]string, map[string][]Method) {
	by := map[string][]Method{}
	for _, m := range methods {
		by[m.Group] = append(by[m.Group], m)
	}
	groups := make([]string, 0, len(by))
	for g := range by {
		groups = append(groups, g)
		sort.Slice(by[g], func(i, j int) bool { return by[g][i].Leaf < by[g][j].Leaf })
	}
	sort.Strings(groups)
	return groups, by
}
