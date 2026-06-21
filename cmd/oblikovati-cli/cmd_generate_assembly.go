// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"oblikovati.org/model/benchgen"
	"oblikovati.org/model/doc"
	"oblikovati.org/perf/benchprof"
	"oblikovati.org/persistence"
)

// cmdGenerateAssembly synthesizes a large, deeply-nested benchmark assembly from a
// named profile and (by default) writes the .obk set under --out, printing the realized
// counts. It is the fixture source for the M34 large-assembly benchmarks:
//
//	oblikovati-cli generate-assembly --profile auto30k --out car30k
//	oblikovati-cli info car30k/asm/root_000000.oad
func cmdGenerateAssembly(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("generate-assembly", flag.ContinueOnError)
	fs.SetOutput(out)
	profileName := fs.String("profile", "auto30k", "assembly profile: auto30k|auto1m")
	outDir := fs.String("out", "", "output directory for the generated .obk set (required)")
	save := fs.Bool("save", true, "write the .obk files (use --save=false to measure generation only)")
	// This command's flags take values (unlike the boolean-only flags partitionFlags
	// assumes), so parse with the standard flag package directly.
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("generate-assembly: unexpected argument(s) %v; flags only (see `oblikovati-cli help`)", fs.Args())
	}
	if *outDir == "" {
		return fmt.Errorf("generate-assembly: --out <dir> is required")
	}
	profile, err := benchgen.ProfileByName(*profileName)
	if err != nil {
		return err
	}
	return generateAssembly(profile, *outDir, *save, out)
}

// generateAssembly builds the profile into a fresh workspace, optionally saves every
// document, and reports the result — split from flag parsing so it is unit-testable.
func generateAssembly(profile benchgen.Profile, outDir string, save bool, out io.Writer) error {
	run, err := benchprof.Start("generate-" + profile.Name)
	if err != nil {
		return err
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	root, stats, err := benchgen.Generate(ws, outDir, profile)
	if err != nil {
		return err
	}
	if save {
		if err := saveWorkspace(ws); err != nil {
			return err
		}
	}
	summary, err := run.Stop()
	if err != nil {
		return err
	}
	printAssemblyStats(out, stats, root, outDir, save)
	fmt.Fprintf(out, "memory:          %s\n", summary)
	return nil
}

// saveWorkspace writes every open document to disk, creating each document's parent
// directory first (the package store writes atomically beside the file but does not
// create the tree). This is the cold-load fixture: one .obk per unique part plus the
// sub-assemblies.
func saveWorkspace(ws *doc.Workspace) error {
	for _, d := range ws.Documents() {
		dir := filepath.Dir(d.FullFileName())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("generate-assembly: mkdir %q: %w", dir, err)
		}
		if err := ws.Save(d); err != nil {
			return fmt.Errorf("generate-assembly: save %q: %w", d.FullFileName(), err)
		}
	}
	return nil
}

// printAssemblyStats reports the realized shape so the spec match is visible and the
// caller knows which file to open.
func printAssemblyStats(out io.Writer, s benchgen.Stats, root *doc.Document, outDir string, saved bool) {
	fmt.Fprintf(out, "profile:         %s\n", s.Profile)
	fmt.Fprintf(out, "leaf placements: %d\n", s.LeafPlacements)
	fmt.Fprintf(out, "  fasteners:     %d\n", s.PerTier[benchgen.Fastener])
	fmt.Fprintf(out, "  brackets:      %d\n", s.PerTier[benchgen.Bracket])
	fmt.Fprintf(out, "  machined:      %d\n", s.PerTier[benchgen.Machined])
	fmt.Fprintf(out, "  systems:       %d\n", s.PerTier[benchgen.System])
	fmt.Fprintf(out, "unique meshes:   %d\n", s.UniqueMeshes)
	fmt.Fprintf(out, "sub-assemblies:  %d\n", s.SubAssemblies)
	fmt.Fprintf(out, "documents:       %d\n", s.Documents)
	fmt.Fprintf(out, "tree depth:      %d\n", s.Depth)
	if saved {
		fmt.Fprintf(out, "wrote %d documents under %s\n", s.Documents, outDir)
		fmt.Fprintf(out, "open with: oblikovati-cli info %s\n", root.FullFileName())
	} else {
		fmt.Fprintln(out, "(not saved; --save=false)")
	}
}
