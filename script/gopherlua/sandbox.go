// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import lua "github.com/yuin/gopher-lua"

// deniedGlobals are the base-library names stripped after OpenBase so a script can
// neither load external code, touch the OS/filesystem, nor reach into the VM:
//   - load/loadstring/loadfile/dofile/require: no code loading (ADR-0028 §2).
//   - collectgarbage: we control GC, not the script.
//   - print: replaced by the captured sink in registerPrint.
//   - module/newproxy: legacy/escape-prone base entries we don't expose.
var deniedGlobals = []string{
	"load", "loadstring", "loadfile", "dofile", "require",
	"collectgarbage", "print", "module", "newproxy",
}

// openSafeLibs registers ONLY the deterministic, no-I/O standard libraries on L:
// base (then stripped), table, string, math. os, io, debug, package/require, and any
// FFI are NEVER opened — they are simply absent from the script environment. This is
// the allow-list half of the sandbox; the deny half is "we never call their opener".
func openSafeLibs(l *lua.LState) {
	openers := []struct {
		name string
		open lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	}
	for _, o := range openers {
		l.Push(l.NewFunction(o.open))
		l.Push(lua.LString(o.name))
		l.Call(1, 0)
	}
	stripDeniedGlobals(l)
}

// stripDeniedGlobals removes each deniedGlobals name from the global table, so the
// dangerous base-library entries opened by OpenBase are gone before any user code runs.
func stripDeniedGlobals(l *lua.LState) {
	for _, name := range deniedGlobals {
		l.SetGlobal(name, lua.LNil)
	}
}

// deniedLibNames lists the standard-library globals a containment test asserts are
// absent from the sandbox. It is the executable form of the deny policy (ADR-0028 §2):
// if any of these becomes reachable, TestSandboxDeniesGlobals fails.
var deniedLibNames = []string{
	"os", "io", "debug", "package", "require",
	"load", "loadstring", "loadfile", "dofile", "collectgarbage",
}
