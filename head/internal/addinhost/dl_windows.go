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
// and the shared openLibrary orchestration lives in dl_shared.go; only these Win32
// calls differ from the dlopen/dlsym path.
//
// GetProcAddress returns FARPROC and LoadLibraryW an HMODULE; the (void*) casts to
// the opaque handle the Go side stores happen here in C, where a function-pointer
// <-> void* cast is the idiomatic Win32 pattern (mirroring how dlsym hands back a
// void* on Unix). There is deliberately NO FreeLibrary wrapper: a loaded Go c-shared
// add-in cannot be safely unmapped (see closeNativeLibrary below).

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

// openNativeLibrary LoadLibrary's path — the Win32 loader primitive behind the shared
// openLibrary in dl_shared.go. The path is passed as UTF-16 via LoadLibraryW so
// non-ASCII add-in paths load correctly.
func openNativeLibrary(path string) (unsafe.Pointer, error) {
	wpath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("addinhost: LoadLibrary path %q not encodable as UTF-16: %w", path, err)
	}
	h := C.obk_load_library((*C.wchar_t)(unsafe.Pointer(wpath)))
	if h == nil {
		return nil, fmt.Errorf("addinhost: LoadLibrary %q: %s", path, lastError())
	}
	return h, nil
}

// lookupOptionalSymbol resolves a symbol the contract marks optional; nil means absent.
func lookupOptionalSymbol(h unsafe.Pointer, name string) unsafe.Pointer {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	return C.obk_get_proc(h, cn)
}

// lookupSymbol resolves a required symbol, returning an error naming the symbol and the
// GetLastError text if absent.
func lookupSymbol(h unsafe.Pointer, name string) (unsafe.Pointer, error) {
	if p := lookupOptionalSymbol(h, name); p != nil {
		return p, nil
	}
	return nil, fmt.Errorf("addinhost: GetProcAddress %q: %s", name, lastError())
}

// closeNativeLibrary intentionally leaves the module RESIDENT — it does NOT call
// FreeLibrary, so the shared close() in dl_shared.go returns nil on Windows. This
// mirrors the Unix host's own lifetime rule (cmd/oblikovati-head/addins.go stop(): "It
// does NOT dlclose the libraries: a Go c-shared keeps runtime threads (sysmon/GC)
// alive, and unmapping its code ... crashes the host"). On Windows that hazard is
// sharper: FreeLibrary on a live Go c-shared DLL unmaps its code while its runtime's
// background threads are still executing, faulting the process with an access violation
// (0xc0000005). Keeping the module mapped for the process lifetime (the OS reclaims it
// on exit) is the safe, production-matching behavior; reload uses the same
// copy-and-coexist strategy as Unix, never an in-process unload.
func closeNativeLibrary(unsafe.Pointer) error { return nil }

// lastError returns the text of the most recent Win32 loader failure captured in C.
func lastError() string { return C.GoString(C.obk_last_message()) }
