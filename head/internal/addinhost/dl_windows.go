//go:build windows

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

/*
#include <windows.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdio.h>

// Windows loader primitives — the only OS-specific part of the add-in loader. The
// resolved symbols are invoked through the portable trampolines in
// include/addin_trampolines.h (shared with dl_unix.go, pulled in via dl_shared.go),
// so only these Win32 calls differ from the dlopen/dlsym path.
//
// GetProcAddress returns FARPROC and LoadLibraryW an HMODULE; the (void*) casts to
// the opaque handle the Go side stores happen here in C, where a function-pointer
// <-> void* cast is the idiomatic Win32 pattern (mirroring how dlsym hands back a
// void* on Unix). There is deliberately NO FreeLibrary wrapper: a loaded Go c-shared
// add-in cannot be safely unmapped (see close() below).

// obk_errbuf holds the last FormatMessage'd error text. It is process-static and
// overwritten on each failure, so the Go caller reads it (into a Go error) right
// after the failing call. Loads are serialized in practice; dlerror() on Unix has
// the same single-buffer contract.
static char obk_errbuf[512];

static void obk_capture_error(void) {
	unsigned long code = GetLastError();
	DWORD n = FormatMessageA(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
	                         NULL, code, 0, obk_errbuf, (DWORD)sizeof(obk_errbuf), NULL);
	if (n == 0) {
		snprintf(obk_errbuf, sizeof(obk_errbuf), "GetLastError=%lu", code);
		return;
	}
	// FormatMessage appends a trailing CRLF; trim it so the Go error reads cleanly.
	while (n > 0 && (obk_errbuf[n-1] == '\n' || obk_errbuf[n-1] == '\r')) obk_errbuf[--n] = 0;
}

static const char* obk_last_message(void) { return obk_errbuf; }

static void* obk_load_library(const wchar_t* path) {
	HMODULE h = LoadLibraryW(path);
	if (h == NULL) obk_capture_error();
	return (void*)h;
}

static void* obk_get_proc(void* h, const char* name) {
	FARPROC p = GetProcAddress((HMODULE)h, name);
	if (p == NULL) obk_capture_error();
	return (void*)p;
}
*/
import "C"

import (
	"fmt"
	"syscall"
	"unsafe"
)

// openLibrary LoadLibrary's path and resolves the required ObkAddIn* exports into a
// sharedLib (dl_shared.go holds the portable export-invocation code; this file holds
// only the Win32 loader primitives). The path is passed as UTF-16 via LoadLibraryW so
// non-ASCII add-in paths load correctly.
func openLibrary(path string) (addInLib, error) {
	wpath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("addinhost: LoadLibrary path %q not encodable as UTF-16: %w", path, err)
	}
	h := C.obk_load_library((*C.wchar_t)(unsafe.Pointer(wpath)))
	if h == nil {
		return nil, fmt.Errorf("addinhost: LoadLibrary %q: %s", path, lastError())
	}
	l := &sharedLib{handle: h, path: path}
	syms := []struct {
		name string
		dst  *unsafe.Pointer
	}{
		{"ObkAddInId", &l.idSym},
		{"ObkAddInManifest", &l.manSym},
		{"ObkAddInActivate", &l.actSym},
		{"ObkAddInDeactivate", &l.deactSym},
		{"ObkAddInNotify", &l.notifSym},
		{"ObkFree", &l.freeSym},
	}
	for _, s := range syms {
		p, err := resolve(h, s.name)
		if err != nil {
			// Deliberately do NOT FreeLibrary the partially-resolved module: once loaded,
			// its Go runtime has already spawned threads, and unmapping it here would race
			// them into a crash exactly as close() explains. A malformed add-in is rare;
			// leaking its mapping is strictly safer than crashing the host.
			return nil, err
		}
		*s.dst = p
	}
	// Automation is an optional export: absence just means hasAutomation() is false.
	l.autoSym = resolveOptional(h, "ObkAddInAutomation")
	// The version exports are resolved leniently so a missing one does not abort the
	// whole load; the compatibility gate (loader.go) turns their absence into a clear
	// "cannot verify compatibility" skip rather than a GetProcAddress error.
	l.majorSym = resolveOptional(h, "ObkAddInApiMajor")
	l.minorSym = resolveOptional(h, "ObkAddInApiMinor")
	return l, nil
}

// resolveOptional looks up a symbol the contract marks optional; nil means absent.
func resolveOptional(h unsafe.Pointer, name string) unsafe.Pointer {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	return C.obk_get_proc(h, cn)
}

// resolve looks up a required symbol, returning a descriptive error (naming the
// symbol and the GetLastError text) if absent.
func resolve(h unsafe.Pointer, name string) (unsafe.Pointer, error) {
	p := resolveOptional(h, name)
	if p == nil {
		return nil, fmt.Errorf("addinhost: GetProcAddress %q: %s", name, lastError())
	}
	return p, nil
}

// close leaves the shared library RESIDENT — it does not FreeLibrary. This mirrors the
// Unix host's own lifetime rule (cmd/oblikovati-head/addins.go stop(): "It does NOT
// dlclose the libraries: a Go c-shared keeps runtime threads (sysmon/GC) alive, and
// unmapping its code ... crashes the host"). On Windows that hazard is sharper: calling
// FreeLibrary on a live Go c-shared DLL unmaps its code while its runtime's background
// threads are still executing, which faults the process with an access violation
// (0xc0000005). So the safe, production-matching behavior is to keep the module mapped
// for the process lifetime; the OS reclaims it on exit. Reload therefore follows the
// same copy-and-coexist strategy as Unix (a fresh LoadLibrary of a replacement copy),
// never an in-process unload. Returns nil because there is nothing to fail.
func (l *sharedLib) close() error { return nil }

// lastError returns the text of the most recent Win32 loader failure captured in C.
func lastError() string { return C.GoString(C.obk_last_message()) }
