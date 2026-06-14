// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// submitCommandLine feeds one line to the Command Window's REPL (M26) and returns what it
// produced: the scrollback lines this submission added, the active command's next prompt,
// and whether more input is awaited. A command-line error (e.g. an unknown command) is
// returned in the result's Error field, not as a transport error, so callers — add-ins and
// MCP tools — see it inline exactly as the UI does.
func submitCommandLine(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.SubmitCommandLineArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	cl := s.CommandLine()
	before := cl.Scrollback().Len()
	submitErr := cl.Submit(s, in.Line)
	res := wire.CommandLineResult{
		Output:   newScrollbackText(cl, before),
		Prompt:   cl.Prompt(s),
		Awaiting: cl.Awaiting(s),
	}
	if submitErr != nil {
		res.Error = submitErr.Error()
	}
	return json.Marshal(res)
}

// newScrollbackText returns the text of the scrollback lines appended since index before.
func newScrollbackText(cl *app.CommandLine, before int) []string {
	lines := cl.Scrollback().Lines()
	if before >= len(lines) {
		return nil
	}
	out := make([]string, 0, len(lines)-before)
	for _, l := range lines[before:] {
		out = append(out, l.Text)
	}
	return out
}
