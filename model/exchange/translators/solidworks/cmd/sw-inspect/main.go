// SPDX-License-Identifier: GPL-2.0-only

// Command sw-inspect extracts a decoded stream from a .SLDPRT/.SLDASM (either container format)
// for inspection: `sw-inspect <file> [stream]`. With no stream it lists them. The bytes written
// are post-decode (format B inflated), i.e. the same for both formats.
package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"oblikovati.org/model/exchange/translators/solidworks/sldprt"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run inspects the .SLDPRT named by args[0]: with no second arg it lists streams; -sketches and
// -features dump the decoded model; any other second arg writes that stream's decoded bytes.
func run(args []string, out io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sw-inspect <file> [stream]")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	doc, err := sldprt.Open(data)
	if err != nil {
		return err
	}
	if len(args) < 2 {
		return listStreams(doc, out)
	}
	switch args[1] {
	case "-sketches":
		return dumpSketches(doc, out)
	case "-features":
		return dumpFeatures(doc, out)
	}
	b, err := doc.Stream(args[1])
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	return err
}

// listStreams prints every stream name and its decoded byte length, sorted by name.
func listStreams(doc *sldprt.Document, out io.Writer) error {
	names := doc.Streams()
	sort.Strings(names)
	for _, n := range names {
		b, _ := doc.Stream(n)
		fmt.Fprintf(out, "%-40s %8d\n", n, len(b))
	}
	return nil
}

// dumpSketches prints one line per decoded sketch with its entity counts and the exact-topology flag.
func dumpSketches(doc *sldprt.Document, out io.Writer) error {
	for i, s := range doc.Sketches() {
		fmt.Fprintf(out, "sk%-2d pts=%d lines=%d arcs=%d circles=%d ellipses=%d splines=%d exact=%v\n",
			i, len(s.Points), len(s.Lines), len(s.Arcs), len(s.Circles), len(s.Ellipses), len(s.Splines), s.Exact)
	}
	return nil
}

// dumpFeatures prints the live feature tree and how many of its nodes are sketches versus decoded.
func dumpFeatures(doc *sldprt.Document, out io.Writer) error {
	tree := doc.FeatureTree()
	nsk := 0
	for _, f := range tree {
		if f.Kind == sldprt.KindSketch {
			nsk++
		}
	}
	fmt.Fprintf(out, "# tree=%d treeSketches=%d decodedSketches=%d\n", len(tree), nsk, len(doc.Sketches()))
	for i, f := range tree {
		fmt.Fprintf(out, "%2d  %-20s %s\n", i, f.Kind, f.Name)
	}
	return nil
}
