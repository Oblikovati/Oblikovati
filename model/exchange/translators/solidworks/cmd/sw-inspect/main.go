// SPDX-License-Identifier: GPL-2.0-only

// Command sw-inspect extracts a decoded stream from a .SLDPRT/.SLDASM (either container format)
// for inspection: `sw-inspect <file> [stream]`. With no stream it lists them. The bytes written
// are post-decode (format B inflated), i.e. the same for both formats.
package main

import (
	"fmt"
	"os"
	"sort"

	"oblikovati.org/model/exchange/translators/solidworks/sldprt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sw-inspect <file> [stream]")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	must(err)
	doc, err := sldprt.Open(data)
	must(err)
	if len(os.Args) < 3 {
		names := doc.Streams()
		sort.Strings(names)
		for _, n := range names {
			b, _ := doc.Stream(n)
			fmt.Printf("%-40s %8d\n", n, len(b))
		}
		return
	}
	if os.Args[2] == "-features" {
		for i, f := range doc.FeatureTree() {
			fmt.Printf("%2d  %-20s %s\n", i, f.Kind, f.Name)
		}
		return
	}
	b, err := doc.Stream(os.Args[2])
	must(err)
	os.Stdout.Write(b)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
