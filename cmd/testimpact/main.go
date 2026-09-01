// SPDX-License-Identifier: GPL-2.0-only

// Command testimpact prints the packages whose tests the current change set can
// affect, one import path per line, for `make test-impacted` to feed to `go test`.
//
//	go run ./cmd/testimpact -base origin/develop
//
// It prints nothing when no package owns the change (a docs-only edit), so the
// caller must treat empty output as "no tests to run", not as an error.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	for _, p := range paths {
		fmt.Fprintln(w, p)
	}
	return nil
}
