// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"oblikovati/addin/opregistry"
	"oblikovati/addin/router"
	"oblikovati/app"
	"oblikovati/persistence"
	"oblikovati/script"
	"oblikovati/script/bridge"
	"oblikovati/script/gopherlua"
	"oblikovati/script/runner"
)

// cmdScript runs a Lua script against a headless session. It links no native code, so
// it stays CGO-free (ADR-0008); the script drives the model through the same wire
// surface add-ins and the MCP bridge use, sandboxed (ADR-0028):
//
//	oblikovati-cli script run build.lua
//	oblikovati-cli script run build.lua --doc in.obk --save out.obk
func cmdScript(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "run" {
		return fmt.Errorf("script: expected `run <file.lua>`, got %q", joinArgs(args))
	}
	return cmdScriptRun(args[1:], out)
}

// cmdScriptRun parses the `script run` operands/flags, builds a session + in-proc
// caller, runs the file under the CLI limits, prints script output to out and errors to
// stderr, and returns a non-nil error (→ non-zero exit) when the script fails.
func cmdScriptRun(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("script run", flag.ContinueOnError)
	fs.SetOutput(out)
	docPath := fs.String("doc", "", "open an existing .obk document before running")
	savePath := fs.String("save", "", "save the active document to this path after a successful run")
	// The file path is the sole positional and must come first; flags (which take
	// values) follow it. flag.Parse stops at the first non-flag, so parse from arg[1:].
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("script run: expected <file.lua> as the first argument, got %q", joinArgs(args))
	}
	file := args[0]
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("script run: unexpected extra argument %q", fs.Arg(0))
	}
	src, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("script run: read %q: %w", file, err)
	}
	return runScriptFile(string(src), *docPath, *savePath, out)
}

// runScriptFile wires a store-backed session and the in-proc router caller, optionally
// opens a document, runs source, then optionally saves. Script print() goes to out; a
// script failure becomes the returned error.
func runScriptFile(source, docPath, savePath string, out io.Writer) error {
	session := app.NewSessionWithStore(persistence.NewPackageStore())
	if docPath != "" {
		if _, err := session.OpenDocument(docPath); err != nil {
			return fmt.Errorf("script run: open %q: %w", docPath, err)
		}
	}
	res := runOnSession(session, source, out)
	if res.Err != nil {
		return fmt.Errorf("script run: %w", res.Err)
	}
	return saveIfRequested(session, savePath, out)
}

// runOnSession builds the runner over a DirectCaller (CLI: synchronous, no UI to
// protect) and runs source under the CLI limits, forwarding print() to out.
func runOnSession(session *app.Session, source string, out io.Writer) script.Result {
	rtr := router.New(opregistry.Default())
	caller := bridge.NewDirectCaller(rtr.Handle, session)
	run := runner.New(gopherlua.New(), caller, rtr.Methods)
	res, err := run.Run(context.Background(), source, runner.DefaultCLILimits, func(line string) {
		fmt.Fprintln(out, line)
	})
	if err != nil { // ErrBusy can't happen on a fresh runner; surface defensively.
		return script.Result{Err: err}
	}
	return res
}

// saveIfRequested writes the active document to savePath when set, so a script that
// builds a model can persist it for an e2e fixture.
func saveIfRequested(session *app.Session, savePath string, out io.Writer) error {
	if savePath == "" {
		return nil
	}
	if err := session.SaveActiveDocumentAs(savePath); err != nil {
		return fmt.Errorf("script run: save %q: %w", savePath, err)
	}
	fmt.Fprintf(out, "saved active document to %s\n", savePath)
	return nil
}

// joinArgs renders args for an error message; empty becomes "<none>".
func joinArgs(args []string) string {
	if len(args) == 0 {
		return "<none>"
	}
	return args[0]
}
