// SPDX-License-Identifier: GPL-2.0-only

// Command oblikovati-cli is the headless entry point — it creates, inspects, and
// re-saves document packages without a window. It links no native code, so it
// builds and runs with CGO_ENABLED=0 (architecture/ADR-0008). Its primary job today
// is generating document test-case fixtures for end-to-end tests, each stamped with
// its per-kind extension (ADR-0034):
//
//	oblikovati-cli new part fixtures/bracket.opd
//	oblikovati-cli info fixtures/bracket.opd
//	oblikovati-cli save-as fixtures/bracket.opd fixtures/copy.opd
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run dispatches argv (without the program name) to a subcommand, writing
// user-facing output to out. It returns a non-nil error on bad usage or IO failure;
// main turns that into a non-zero exit. Keeping the dispatch pure (no os.Exit, an
// injected writer) is what makes the subcommands table-testable.
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("oblikovati-cli: no command\n%s", usage)
	}
	switch args[0] {
	case "new":
		return cmdNew(args[1:], out)
	case "open":
		return cmdOpen(args[1:], out)
	case "info":
		return cmdInfo(args[1:], out)
	case "save-as":
		return cmdSaveAs(args[1:], out)
	case "import":
		return cmdImport(args[1:], out)
	case "export":
		return cmdExport(args[1:], out)
	case "export-flat":
		return cmdExportFlat(args[1:], out)
	case "script":
		return cmdScript(args[1:], out)
	case "generate-assembly":
		return cmdGenerateAssembly(args[1:], out)
	case "version":
		return cmdVersion(out)
	case "check-updates":
		return cmdCheckUpdates(out)
	case "help", "-h", "--help":
		fmt.Fprintln(out, usage)
		return nil
	default:
		return fmt.Errorf("oblikovati-cli: unknown command %q (want new|open|info|save-as|import|export|export-flat|script|generate-assembly|version|check-updates)", args[0])
	}
}

// usage is the one-screen command summary printed on `help` or a usage error.
const usage = `oblikovati-cli — headless document tool

usage:
  oblikovati-cli new <part|assembly|drawing|presentation> <path> [--seed]
  oblikovati-cli open <path>
  oblikovati-cli info <path> [--json]
  oblikovati-cli save-as <src> <dst>
  oblikovati-cli import <mesh-file.stl|.obj|.3mf> <dst.opd>
  oblikovati-cli export <src.opd> <mesh-file.stl|.obj|.3mf> [low|medium|high]
  oblikovati-cli export-flat <src.opd> <out.dxf> [r2000|r2018]
  oblikovati-cli script run <file.lua> [--doc in.opd] [--save out.opd]
  oblikovati-cli generate-assembly --profile <auto30k|auto1m> --out <dir> [--save=false]
  oblikovati-cli version
  oblikovati-cli check-updates`
