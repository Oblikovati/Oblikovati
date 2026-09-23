// SPDX-License-Identifier: GPL-2.0-only

// Command testimpact prints the packages whose tests the current change set can
// affect, one import path per line, for `make test-impacted` to feed to `go test`.
//
//	go run ./cmd/testimpact -base origin/develop
//
// It prints nothing when no package owns the change (a docs-only edit), so the
// caller must treat empty output as "no tests to run", not as an error.
//
// Packages git IGNORES are left out: `go list ./...` walks experiments/, which .gitignore
// excludes and which holds by design unfinished code, so a half-written experiment on one
// developer's disk failed every local run of a gate about code being committed while CI — which
// never receives those files — passed (#3557).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"oblikovati.org/test-utilities/testimpact"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "testimpact:", err)
		os.Exit(1)
	}
}

// run parses args and writes the impacted import paths, one per line, to w.
func run(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("testimpact", flag.ContinueOnError)
	fs.SetOutput(w)
	base := fs.String("base", "origin/develop",
		"revision to compare against; empty compares the working tree only")
	root := fs.String("root", ".", "module root to analyse")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return printImpacted(w, *root, *base)
}

// printImpacted resolves the module root, runs the selection against the real git
// working copy and package graph, and writes the result to w.
// modulePath is this module's import prefix; an import path minus it is a directory.
const modulePath = "oblikovati.org"

func printImpacted(w io.Writer, root, base string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root %q: %w", root, err)
	}
	sel := testimpact.NewSelector(abs,
		testimpact.NewGoListLoader(abs),
		testimpact.NewGitChanges(abs, base))
	paths, err := sel.Impacted()
	if err != nil {
		return err
	}
	for _, p := range tracked(abs, paths) {
		fmt.Fprintln(w, p)
	}
	return nil
}

// tracked drops the import paths whose directory git ignores. It asks git once, rather than
// matching .gitignore here, so the answer is whatever git itself would say.
func tracked(root string, paths []string) []string {
	dirs := make([]string, 0, len(paths))
	for _, p := range paths {
		dirs = append(dirs, "."+strings.TrimPrefix(p, modulePath))
	}
	// check-ignore exits 1 when nothing matches, which is the common case and not an error.
	cmd := exec.Command("git", append([]string{"check-ignore", "--stdin"}, nil...)...)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(strings.Join(dirs, "\n"))
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return paths // git unavailable or nothing ignored: keep every package
	}
	ignored := make(map[string]bool, 8)
	for _, d := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if d != "" {
			ignored[strings.TrimSuffix(d, "/")] = true
		}
	}
	kept := paths[:0:0]
	for i, p := range paths {
		if !ignored[strings.TrimSuffix(dirs[i], "/")] {
			kept = append(kept, p)
		}
	}
	return kept
}
