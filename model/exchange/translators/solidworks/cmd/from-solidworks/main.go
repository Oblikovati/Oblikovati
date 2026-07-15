// SPDX-License-Identifier: GPL-2.0-only

// Command from-solidworks is the headless entry point for the SolidWorks translator: it reads a
// SolidWorks .SLDPRT part and writes an Oblikovati package (.opd) — global variables become
// parameters and every decoded sketch is emitted. It needs no running SolidWorks and reads both
// container formats (older CFBF and the SolidWorks 2026 native store).
//
// Usage:
//
//	from-solidworks <file.SLDPRT> [-o <out.opd>]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oblikovati.org/model/exchange/translators/solidworks/translate"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "from-solidworks:", err)
		os.Exit(1)
	}
}

func run(args []string, out *os.File) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("expected <file.SLDPRT> as the first argument")
	}
	src := args[0]
	fs := flag.NewFlagSet("from-solidworks", flag.ContinueOnError)
	fs.SetOutput(out)
	dst := fs.String("o", "", "destination .opd package (default: <file>.opd)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	outPath := *dst
	if outPath == "" {
		outPath = strings.TrimSuffix(src, filepath.Ext(src)) + ".opd"
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	warns, err := translate.FromSolidWorks(data, outPath)
	for _, w := range warns {
		fmt.Fprintln(out, "warning:", w)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "wrote", outPath)
	return nil
}
