// SPDX-License-Identifier: GPL-2.0-only

// Command command-manual renders the Command Window's command manual as a GitHub-wiki
// Markdown page (grouped by document type, then by area), from the built-in vocabulary.
// The wiki publish workflow runs it on every merge to develop so the manual never drifts:
//
//	go run ./cmd/command-manual > Command-Manual.md   # write to stdout
//	go run ./cmd/command-manual path/to/Command-Manual.md   # write to a file
package main

import (
	"fmt"
	"os"

	"oblikovati.org/app/cmdline"
)

func main() {
	md := cmdline.DefaultVocabulary().RenderWikiManual()
	if len(os.Args) > 1 {
		if err := os.WriteFile(os.Args[1], []byte(md), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "command-manual: write %s: %v\n", os.Args[1], err)
			os.Exit(1)
		}
		return
	}
	if _, err := os.Stdout.WriteString(md); err != nil {
		fmt.Fprintf(os.Stderr, "command-manual: %v\n", err)
		os.Exit(1)
	}
}
