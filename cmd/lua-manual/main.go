// SPDX-License-Identifier: GPL-2.0-only

// Command lua-manual renders the Lua Scripting wiki page from the public API: the method
// reference comes from api/wire + the api/client mcp:summary annotations (the same source the
// MCP bridge uses), and the worked examples from the bundled script/examples library. The
// wiki publish workflow runs it on every merge so the manual never drifts:
//
//	go run ./cmd/lua-manual > Lua-Scripting.md      # write to stdout
//	go run ./cmd/lua-manual path/to/Lua-Scripting.md   # write to a file
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"oblikovati.org/script/examples"
	"oblikovati.org/script/luadoc"
)

func main() {
	dir, err := apiDir()
	if err != nil {
		fail(err)
	}
	exs, err := loadExamples()
	if err != nil {
		fail(err)
	}
	md, err := luadoc.Generate(dir, exs)
	if err != nil {
		fail(err)
	}
	if len(os.Args) > 1 {
		if err := os.WriteFile(os.Args[1], []byte(md), 0o644); err != nil {
			fail(err)
		}
		return
	}
	if _, err := os.Stdout.WriteString(md); err != nil {
		fail(err)
	}
}

// apiDir resolves the on-disk oblikovati.org/api module root (honoring go.work / the CI
// replace), so the generator parses the same contract the build links against.
func apiDir() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "oblikovati.org/api").Output()
	if err != nil {
		return "", fmt.Errorf("lua-manual: resolve api dir: %w", err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("lua-manual: empty api dir from go list")
	}
	return dir, nil
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

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
