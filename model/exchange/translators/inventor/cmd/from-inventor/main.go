// SPDX-License-Identifier: GPL-2.0-only

// Command from-inventor is the headless entry point for the Inventor translator: it reads
// an Autodesk Inventor .ipt part or .iam assembly and writes an Oblikovati package (.opd).
// For an assembly it also translates the referenced component parts (siblings on disk) and
// places each occurrence at its decoded transform. It needs no running Inventor. (The GUI
// surfaces this as File ▸ Translate ▸ From Inventor.)
//
// Usage:
//
//	from-inventor <file.ipt|file.iam> [-o <out.opd>]
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"oblikovati.org/model/exchange/translators/inventor/translate"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "from-inventor:", err)
		os.Exit(1)
	}
}

func run(args []string, out *os.File) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("expected <file.ipt> as the first argument")
	}
	src := args[0]
	fs := flag.NewFlagSet("from-inventor", flag.ContinueOnError)
	fs.SetOutput(out)
	dst := fs.String("o", "", "destination .opd package (default: <file>.opd)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	isAssembly := strings.HasSuffix(strings.ToLower(src), ".iam")
	target := *dst
	if target == "" {
		target = strings.TrimSuffix(strings.TrimSuffix(src, ".iam"), ".ipt") + ".opd"
	}
	warns, err := runTranslate(src, target, isAssembly)
	for _, w := range warns {
		fmt.Fprintln(out, "warning:", w)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "wrote %s\n", target)
	return nil
}

// runTranslate dispatches to the assembly translator (which resolves component files on
// disk) or the part translator (which works on the file's bytes).
func runTranslate(src, target string, isAssembly bool) ([]string, error) {
	if isAssembly {
		return translate.AssemblyFromInventor(src, target)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	return translate.FromInventor(data, target)
}
