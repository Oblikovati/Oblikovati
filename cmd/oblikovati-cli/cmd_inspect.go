// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"oblikovati/model/doc"
	"oblikovati/persistence"
)

// cmdOpen loads the package at path through the real open flow (the store-backed
// workspace) and reports the reconstructed document's identity — proving the file
// round-trips into a live document, which is what an e2e open test asserts.
func cmdOpen(args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("open: expected <path>, got %d arg(s)", len(args))
	}
	path := args[0]
	ws := doc.NewWorkspace(persistence.NewPackageStore())
	d, err := ws.Open(path, true)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	fmt.Fprintf(out, "opened %s %q from %s\n", d.DocumentType(), d.DisplayName(), path)
	return nil
}

// manifestReport is the JSON shape printed by `info --json`, so e2e tests can parse a
// fixture's identity without scraping plain text.
type manifestReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	Type          string `json:"type"`
	Name          string `json:"name"`
}

// cmdInfo reads the package manifest at path and prints its identity — schema
// version, document kind, display name — without reconstructing a full document.
func cmdInfo(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	fs.SetOutput(out)
	asJSON := fs.Bool("json", false, "print the manifest as a JSON object")
	// Reorder so the --json flag may sit before OR after the path; Go's flag package
	// otherwise stops at the first positional argument.
	flags, operands := partitionFlags(args)
	if err := fs.Parse(flags); err != nil {
		return err
	}
	if len(operands) != 1 {
		return fmt.Errorf("info: expected <path>, got %d arg(s)", len(operands))
	}
	report, err := readManifest(operands[0])
	if err != nil {
		return err
	}
	if *asJSON {
		return json.NewEncoder(out).Encode(report)
	}
	fmt.Fprintf(out, "schema version: %d\ntype: %s\nname: %s\n",
		report.SchemaVersion, report.Type, report.Name)
	return nil
}

// partitionFlags splits args into dash-prefixed flags and bare operands, so a
// subcommand can accept its flag in any position. It assumes flags are booleans (no
// separate value token), which holds for every oblikovati-cli flag today.
func partitionFlags(args []string) (flags, operands []string) {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			continue
		}
		operands = append(operands, a)
	}
	return flags, operands
}

// readManifest opens the package and projects its manifest into a manifestReport,
// translating the persisted DocumentType code into its stable name.
func readManifest(path string) (manifestReport, error) {
	pkg, err := persistence.OpenPackage(path)
	if err != nil {
		return manifestReport{}, fmt.Errorf("info: %w", err)
	}
	m, err := pkg.Manifest()
	if err != nil {
		return manifestReport{}, fmt.Errorf("info: %w", err)
	}
	return manifestReport{
		SchemaVersion: m.SchemaVersion,
		Type:          doc.DocumentType(m.DocumentType).String(),
		Name:          m.DisplayName,
	}, nil
}
