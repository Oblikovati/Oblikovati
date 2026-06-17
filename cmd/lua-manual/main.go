// SPDX-License-Identifier: GPL-2.0-only

// Command lua-manual renders the Lua Scripting wiki page from the public API: the method
// reference comes from api/wire + the api/client mcp:summary annotations (the same source the
// MCP bridge uses), and the worked examples from the bundled script/examples library. The
// wiki publish workflow runs it on every merge so the manual never drifts.
//
// The api module directory is passed in (the caller resolves it with
// `go list -m -f {{.Dir}} oblikovati.org/api`), so this tool launches no subprocess:
//
//	APIDIR=$(go list -m -f '{{.Dir}}' oblikovati.org/api)
//	go run ./cmd/lua-manual "$APIDIR"                       # write to stdout
//	go run ./cmd/lua-manual "$APIDIR" path/to/Lua-Scripting.md   # write to a file
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"oblikovati.org/script/examples"
	"oblikovati.org/script/luadoc"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run renders the manual from the api dir given as the first argument and writes it to the
// second argument (a file) or to stdout when no file is given. Kept separate from main so it
// is testable without a subprocess or os.Exit.
func run(args []string, stdout io.Writer) error {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("lua-manual: usage: lua-manual <apiDir> [outPath]")
	}
	exs, err := loadExamples()
	if err != nil {
		return err
	}
	md, err := luadoc.Generate(args[0], exs)
	if err != nil {
		return err
	}
	if len(args) >= 2 && args[1] != "" {
		return os.WriteFile(args[1], []byte(md), 0o644)
	}
	_, err = io.WriteString(stdout, md)
	return err
}

// loadExamples turns the bundled example programs into manual entries, using each program's
// leading comment block as its description.
func loadExamples() ([]luadoc.Example, error) {
	names, err := examples.Names()
	if err != nil {
		return nil, err
	}
	out := make([]luadoc.Example, 0, len(names))
	for _, name := range names {
		src, err := examples.Source(name)
		if err != nil {
			return nil, err
		}
		out = append(out, luadoc.Example{Name: name, Description: leadingComment(src), Source: src})
	}
	return out, nil
}

// leadingComment returns the program's opening "-- " comment block as a single sentence, the
// human description shown above its source.
func leadingComment(src string) string {
	var parts []string
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(t, "--")
		if !ok {
			break
		}
		parts = append(parts, strings.TrimSpace(rest))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
