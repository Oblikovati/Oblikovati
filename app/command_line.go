// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/app/cmdline"
)

// CommandLine is the Command Window's interactive engine (M26): an AutoCAD-style REPL over
// the existing command + tool model. Submitting a line either starts a command (resolved
// through the binding engine's vocabulary) or feeds the active tool one parsed token —
// coordinate, value, or keyword — advancing its prompt. It reuses the pure cmdline parser
// and scrollback; everything that touches the Session lives here so the head and the API
// drive one engine. Relative ("@") coordinates resolve against the previous point.
type CommandLine struct {
	sb        *cmdline.Scrollback
	lastPoint *cmdline.Coord // last absolute point this interaction, for @relative input
}

// newCommandLine returns an engine with a bounded scrollback.
func newCommandLine() *CommandLine { return &CommandLine{sb: cmdline.NewScrollback(0)} }

// CommandLine returns the session's command-line engine, building it on first use.
func (s *Session) CommandLine() *CommandLine {
	if s.cmdLine == nil {
		s.cmdLine = newCommandLine()
	}
	return s.cmdLine
}

// Scrollback exposes the rolling output for the head and the API to render.
func (cl *CommandLine) Scrollback() *cmdline.Scrollback { return cl.sb }

// Prompt returns the question or step the input line is waiting on: a pending command-line
// question (M26 F03) first, then the active tool's current step prompt, else "".
func (cl *CommandLine) Prompt(s *Session) string {
	if spec, ok := s.Prompts().Pending(); ok {
		return spec.Message + " [" + strings.Join(spec.Buttons, "/") + "]"
	}
	ti := s.ActiveTool()
	if ti == nil {
		return ""
	}
	return promptText(s, ti.Tool())
}

// Completions returns autocomplete suggestions for the current input prefix — the command
// names whose word matches (M26). It is empty while a command is mid-interaction (the input
// is then a coordinate/value/answer, not a new command), so the hint list only shows when a
// command word is being typed.
func (cl *CommandLine) Completions(s *Session, prefix string) []string {
	if cl.Awaiting(s) {
		return nil
	}
	return s.Bindings().Completions(prefix)
}

// Awaiting reports whether the engine is waiting on input — a pending question or an active
// tool mid-interaction — so a caller knows more input is expected before the command ends.
func (cl *CommandLine) Awaiting(s *Session) bool {
	if _, ok := s.Prompts().Pending(); ok {
		return true
	}
	return s.ActiveTool() != nil
}

// Submit feeds one line to the engine. A pending command-line question is answered first
// (M26 F03); otherwise, while a tool is active the line is a token for it, and when idle it
// starts a command (an empty idle line repeats the last command).
func (cl *CommandLine) Submit(s *Session, line string) error {
	line = strings.TrimSpace(line)
	if _, ok := s.Prompts().Pending(); ok {
		return cl.answerPrompt(s, line)
	}
	if s.ActiveTool() != nil {
		return cl.feedToken(s, line)
	}
	return cl.startCommand(s, line)
}

// answerPrompt resolves the pending prompt with the typed answer: an empty line accepts the
// default button, otherwise the token is matched against the answer options (keyword/prefix).
// An unrecognised answer re-asks rather than guessing (M26 F03).
func (cl *CommandLine) answerPrompt(s *Session, line string) error {
	spec, _ := s.Prompts().Pending()
	answer, ok := cl.resolveAnswer(spec, line)
	if !ok {
		cl.appendf(cmdline.Warning, "Answer with one of [%s]", strings.Join(spec.Buttons, "/"))
		return nil
	}
	cl.sb.Append(answer, cmdline.Echo)
	if err := s.AnswerPrompt(spec.ID, answer, false); err != nil {
		cl.appendf(cmdline.Error, "%v", err)
		return err
	}
	return nil
}

// resolveAnswer maps a typed line to one of a prompt's buttons: "" accepts the default,
// else a case-insensitive exact/prefix match.
func (cl *CommandLine) resolveAnswer(spec PromptSpec, line string) (string, bool) {
	if line == "" {
		return spec.Buttons[spec.Default], true
	}
	return cmdline.MatchKeyword(line, spec.Buttons)
}

// startCommand resolves the first word to an action and dispatches it, then feeds any
// inline tokens (so "LINE 0,0 10,0" works in one submission). An empty line repeats the
// last command (AutoCAD's Enter-to-repeat).
func (cl *CommandLine) startCommand(s *Session, line string) error {
	fields := cmdline.Fields(line)
	if len(fields) == 0 {
		return cl.repeatLast(s)
	}
	word := fields[0]
	cl.sb.Append(word, cmdline.Echo)
	actionID, ok := cl.resolveWord(s, word)
	if !ok {
		cl.appendf(cmdline.Error, "Unknown command: %q", word)
		return fmt.Errorf("cmdline: unknown command %q", word)
	}
	cl.sb.RecordCommand(word)
	return cl.dispatch(s, actionID, fields[1:])
}

// resolveWord maps a typed command word to an action: a typed alias / AutoCAD vocabulary
// word first, then — for a single character — that key's keyboard shortcut, so a one-letter
// shortcut still runs when typed and committed with Enter (e.g. "V" → toggle visibility).
func (cl *CommandLine) resolveWord(s *Session, word string) (string, bool) {
	b := s.Bindings()
	if actionID, ok := b.ResolveAlias(word); ok {
		return actionID, true
	}
	if len([]rune(word)) == 1 {
		return b.ResolveChord(types.KeyChord{Key: word})
	}
	return "", false
}

// RunChord runs an action triggered by a keyboard chord (Ctrl+S, Ctrl+Z, …) as if it were
// typed: it echoes the action's canonical command word, then dispatches it — so a chord and
// the equivalent typed command behave and read identically (M26 F05).
func (cl *CommandLine) RunChord(s *Session, actionID string) error {
	word := s.Bindings().CanonicalWord(actionID)
	cl.sb.Append(word, cmdline.Echo)
	cl.sb.RecordCommand(word)
	return cl.dispatch(s, actionID, nil)
}

// dispatch runs the resolved action, shows the started tool's prompt, and feeds any inline
// tokens that followed the command word.
func (cl *CommandLine) dispatch(s *Session, actionID string, inline []string) error {
	cl.reset()
	if err := s.Bindings().Dispatch(actionID, s); err != nil {
		cl.appendf(cmdline.Error, "%v", err)
		return err
	}
	if ti := s.ActiveTool(); ti != nil {
		if ct, ok := ti.Tool().(continuousCommandTool); ok {
			ct.EnableContinuous() // command-line LINE chains, AutoCAD-style (viewport stays single-shape)
		}
	}
	cl.showPrompt(s)
	for _, tok := range inline {
		if s.ActiveTool() == nil {
			break
		}
		if err := cl.feedToken(s, tok); err != nil {
			return err
		}
	}
	return nil
}

// repeatLast re-invokes the most recently submitted command (its verb only).
func (cl *CommandLine) repeatLast(s *Session) error {
	last, ok := cl.sb.LastCommand()
	if !ok {
		return nil
	}
	return cl.startCommand(s, last)
}

// feedToken parses one token for the active tool and applies it. An empty token finishes
// the command (Enter); a tool that takes no typed input is reported, not driven.
func (cl *CommandLine) feedToken(s *Session, raw string) error {
	if raw == "" {
		return cl.finish(s)
	}
	tool := s.ActiveTool().Tool()
	driven, ok := tool.(CommandDriven)
	if !ok {
		cl.appendf(cmdline.Warning, "%s takes input in the viewport", s.ActiveTool().Name())
		return nil
	}
	tok, ok := cl.parseToken(s, tool, raw)
	if !ok {
		cl.appendf(cmdline.Error, "Unrecognized input: %q", raw)
		return fmt.Errorf("cmdline: unrecognized input %q", raw)
	}
	return cl.apply(s, driven, tok)
}

// parseToken classifies a raw token against the current step: a matching bracketed keyword
// first (for an option-bearing step), then a coordinate, then a bare value.
func (cl *CommandLine) parseToken(s *Session, tool Tool, raw string) (CommandToken, bool) {
	if co, ok := tool.(commandOptioned); ok {
		if opts := co.CommandOptions(s); len(opts) > 0 {
			if kw, matched := cmdline.MatchKeyword(raw, opts); matched {
				return CommandToken{Kind: KeywordToken, Keyword: kw}, true
			}
		}
	}
	if c, ok := cmdline.ParseCoord(raw); ok {
		return CommandToken{Kind: CoordToken, Coord: cl.resolveRelative(c)}, true
	}
	if v, ok := cmdline.ParseDistance(raw); ok {
		return CommandToken{Kind: ValueToken, Value: v}, true
	}
	return CommandToken{}, false
}

// apply submits a parsed token, then auto-finishes when the tool is ready and the step
// completes the command (a geometry tool that auto-commits, or a value that satisfies it).
func (cl *CommandLine) apply(s *Session, driven CommandDriven, tok CommandToken) error {
	if err := driven.SubmitToken(s, tok); err != nil {
		cl.appendf(cmdline.Error, "%v", err)
		return err
	}
	if cl.shouldFinish(s, tok) {
		return cl.finish(s)
	}
	cl.showPrompt(s)
	return nil
}

// shouldFinish decides whether to commit after a token: a tool that signals it is done
// (tokenFinisher, e.g. LINE's Close), a fixed-arity geometry tool that has its clicks
// (AutoCommitTool), or a value-driven feature now ready to build. A continuous chain reports
// neither until Close, so it keeps collecting points until an explicit Enter (see finish).
func (cl *CommandLine) shouldFinish(s *Session, tok CommandToken) bool {
	ti := s.ActiveTool()
	if ti == nil || !ti.Tool().CanCommit() {
		return false
	}
	if tf, ok := ti.Tool().(tokenFinisher); ok {
		return tf.FinishAfterToken()
	}
	if ac, ok := ti.Tool().(AutoCommitTool); ok && ac.AutoCommits() {
		return true
	}
	return tok.Kind == ValueToken
}

// finish commits the active tool when it is ready, else cancels it, and reports the
// outcome on the command line.
func (cl *CommandLine) finish(s *Session) error {
	ti := s.ActiveTool()
	if ti == nil {
		return nil
	}
	name := ti.Name()
	cl.reset()
	if !ti.Tool().CanCommit() {
		s.CancelTool()
		cl.appendf(cmdline.Info, "%s cancelled", name)
		return nil
	}
	if err := s.OK(); err != nil {
		cl.appendf(cmdline.Error, "%v", err)
		return err
	}
	cl.appendf(cmdline.Info, "%s created", name)
	return nil
}

// resolveRelative turns a relative ("@") coordinate into an absolute one against the last
// point of this interaction, and records the result as the new last point.
func (cl *CommandLine) resolveRelative(c cmdline.Coord) cmdline.Coord {
	abs := c
	if c.Relative && cl.lastPoint != nil {
		abs = cmdline.Coord{X: cl.lastPoint.X + c.X, Y: cl.lastPoint.Y + c.Y, Z: cl.lastPoint.Z + c.Z}
	}
	abs.Relative = false
	cl.lastPoint = &abs
	return abs
}

// showPrompt appends the active tool's current step prompt to the scrollback.
func (cl *CommandLine) showPrompt(s *Session) {
	if ti := s.ActiveTool(); ti != nil {
		if p := promptText(s, ti.Tool()); p != "" {
			cl.sb.Append(p, cmdline.Prompt)
		}
	}
}

// reset clears the per-interaction relative-coordinate anchor.
func (cl *CommandLine) reset() { cl.lastPoint = nil }

// appendf appends a formatted line at the given severity.
func (cl *CommandLine) appendf(sev cmdline.Severity, format string, args ...any) {
	cl.sb.Append(fmt.Sprintf(format, args...), sev)
}

// promptText renders a tool's current step prompt — its existing [Prompted] text — with any
// bracketed keyword options appended, or "" when the tool has no prompt.
func promptText(s *Session, t Tool) string {
	p, ok := t.(Prompted)
	if !ok {
		return ""
	}
	text := p.Prompt(s)
	if co, ok := t.(commandOptioned); ok {
		if opts := co.CommandOptions(s); len(opts) > 0 {
			text += " [" + strings.Join(opts, "/") + "]"
		}
	}
	return text
}
