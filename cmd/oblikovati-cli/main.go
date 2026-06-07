// SPDX-License-Identifier: GPL-2.0-only

// Command oblikovati-cli is the headless entry point — it creates, inspects, and
// re-saves .obk document packages without a window. It links no native code, so it
// builds and runs with CGO_ENABLED=0 (architecture/ADR-0008). Its primary job today
// is generating .obk test-case fixtures for end-to-end tests:
//
//	oblikovati-cli new part fixtures/bracket.obk
//	oblikovati-cli info fixtures/bracket.obk
//	oblikovati-cli save-as fixtures/bracket.obk fixtures/copy.obk
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
	case "script":
		return cmdScript(args[1:], out)
	case "version":
		return cmdVersion(out)
	case "help", "-h", "--help":
		fmt.Fprintln(out, usage)
		return nil
	default:
		return fmt.Errorf("oblikovati-cli: unknown command %q (want new|open|info|save-as|import|export|script|version)", args[0])
	}
}

// usage is the one-screen command summary printed on `help` or a usage error.
const usage = `oblikovati-cli — headless .obk document tool

usage:
  oblikovati-cli new <part|assembly|drawing|presentation> <path> [--seed]
  oblikovati-cli open <path>
  oblikovati-cli info <path> [--json]
  oblikovati-cli save-as <src> <dst>
  oblikovati-cli import <mesh-file.stl|.obj|.3mf> <dst.obk>
  oblikovati-cli export <src.obk> <mesh-file.stl|.obj|.3mf> [low|medium|high]
  oblikovati-cli script run <file.lua> [--doc in.obk] [--save out.obk]
  oblikovati-cli version`
