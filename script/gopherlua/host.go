// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import (
	"strings"

	lua "github.com/yuin/gopher-lua"

	"oblikovati/script"
)

// outputSink accumulates a script's print() output for Result.Stdout. It is also
// forwarded line-by-line to the host's Print callback (the console/CLI) as it arrives.
type outputSink struct{ b strings.Builder }

func (o *outputSink) String() string { return o.b.String() }

// registerPrint installs a captured print(...) that joins its arguments with tabs and a
// trailing newline (Lua's print semantics), appends to the sink, and forwards each line
// to host (the console/CLI). The dangerous default print was stripped in the sandbox;
// this is the only print the script sees, so it can never reach a real stdout fd.
func registerPrint(l *lua.LState, host func(string), out *outputSink) {
	l.SetGlobal("print", l.NewFunction(func(s *lua.LState) int {
		line := joinPrintArgs(s)
		out.b.WriteString(line)
		out.b.WriteByte('\n')
		if host != nil {
			host(line)
		}
		return 0
	}))
}

// joinPrintArgs renders print's arguments the way Lua's print does: each argument's
// tostring form, separated by a tab.
func joinPrintArgs(s *lua.LState) string {
	n := s.GetTop()
	parts := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		parts = append(parts, s.ToStringMeta(s.Get(i)).String())
	}
	return strings.Join(parts, "\t")
}

// installOblikovati registers the single host door: a global `oblikovati` table with
// `call(method, argsTable) -> resultTable` and `methods() -> {names}`. All host power
// flows through this one audited table; it reaches the model only via g.Call, which the
// bridge marshals onto the session goroutine (ADR-0028 §3).
func installOblikovati(l *lua.LState, g script.Globals) {
	tb := l.NewTable()
	l.SetField(tb, "call", l.NewFunction(callBinding(g.Call)))
	l.SetField(tb, "methods", l.NewFunction(methodsBinding(g.Methods)))
	l.SetGlobal("oblikovati", tb)
}

// callBinding adapts a script.CallFunc into the Lua `oblikovati.call(method, args)`
// function: it reads the method string and optional args table, marshals the table to
// JSON, invokes call, and converts the JSON result back to a Lua table pushed as the
// single return value. Any conversion or call error is raised as a Lua error so the
// script can pcall it (and an unhandled one surfaces as Result.Err with the line).
func callBinding(call script.CallFunc) lua.LGFunction {
	return func(l *lua.LState) int {
		method := l.CheckString(1)
		args := l.OptTable(2, l.NewTable())
		argsJSON, err := tableToJSON(args)
		if err != nil {
			l.RaiseError("oblikovati.call(%q): %s", method, err.Error())
		}
		resJSON, err := callHost(call, method, argsJSON)
		if err != nil {
			l.RaiseError("oblikovati.call(%q): %s", method, err.Error())
		}
		result, err := jsonToTable(l, resJSON)
		if err != nil {
			l.RaiseError("oblikovati.call(%q): %s", method, err.Error())
		}
		l.Push(result)
		return 1
	}
}

// callHost guards a nil CallFunc (a misconfigured Globals) with a clear error rather
// than a nil-call panic — the message names the offending method per CLAUDE.md.
func callHost(call script.CallFunc, method string, argsJSON []byte) ([]byte, error) {
	if call == nil {
		return nil, errNoCaller(method)
	}
	return call(method, argsJSON)
}

// methodsBinding adapts the Methods provider into `oblikovati.methods()`, returning a
// 1..n array table of the registered method names (empty when no provider is wired).
func methodsBinding(list func() []string) lua.LGFunction {
	return func(l *lua.LState) int {
		tb := l.NewTable()
		if list != nil {
			for _, m := range list() {
				tb.Append(lua.LString(m))
			}
		}
		l.Push(tb)
		return 1
	}
}
