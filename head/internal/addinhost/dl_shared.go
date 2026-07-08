//go:build linux || darwin || windows

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

/*
#cgo CFLAGS: -I${SRCDIR}/include
#include <stdlib.h>
#include <stdint.h>
#include "addin_trampolines.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// sharedLib is a loaded shared-library add-in with its resolved C entry points. The
// handle and symbol pointers are opaque void* (a dlopen handle + dlsym results on
// Unix, an HMODULE + GetProcAddress results on Windows); the OS-specific loader
// primitives that populate them live in dl_unix.go / dl_windows.go, while every
// export invocation below is platform-independent — it goes through the call_*
// trampolines in include/addin_trampolines.h, so it is written once here.
//
// autoSym is the OPTIONAL ObkAddInAutomation export (nil when the add-in has no
// automation surface, M05-F01 #252); freeSym is the add-in's ObkFree, which releases
// the buffers automation hands back across the runtime boundary.
type sharedLib struct {
	handle   unsafe.Pointer
	idSym    unsafe.Pointer
	manSym   unsafe.Pointer
	majorSym unsafe.Pointer
	minorSym unsafe.Pointer
	actSym   unsafe.Pointer
	deactSym unsafe.Pointer
	notifSym unsafe.Pointer
	freeSym  unsafe.Pointer
	autoSym  unsafe.Pointer
	path     string
}

// openLibrary loads the shared library at path and resolves the required ObkAddIn*
// exports, returning a ready sharedLib. The orchestration — required-symbol list,
// resolve loop, optional-symbol resolution, and struct assembly — is identical on
// every OS and lives here once; each platform (dl_unix.go / dl_windows.go) supplies
// only its irreducible primitives (openNativeLibrary / lookupSymbol /
// lookupOptionalSymbol / closeNativeLibrary).
func openLibrary(path string) (addInLib, error) {
	h, err := openNativeLibrary(path)
	if err != nil {
		return nil, err
	}
	l := &sharedLib{handle: h, path: path}
	required := []struct {
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
	for _, s := range required {
		if *s.dst, err = lookupSymbol(h, s.name); err != nil {
			_ = closeNativeLibrary(h) // unwind the partial load (a no-op on Windows)
			return nil, err
		}
	}
	// Automation is an optional export: absence just means hasAutomation() is false.
	l.autoSym = lookupOptionalSymbol(h, "ObkAddInAutomation")
	// The version exports are resolved leniently so a missing one does not abort the
	// whole load; the compatibility gate (loader.go) turns their absence into a clear
	// "cannot verify compatibility" skip rather than a symbol-lookup error.
	l.majorSym = lookupOptionalSymbol(h, "ObkAddInApiMajor")
	l.minorSym = lookupOptionalSymbol(h, "ObkAddInApiMinor")
	return l, nil
}

// close unloads the library through the platform primitive, tagging the path onto any
// failure. On Unix this dlcloses; on Windows closeNativeLibrary deliberately leaves the
// module resident (see dl_windows.go), so this returns nil there.
func (l *sharedLib) close() error {
	if err := closeNativeLibrary(l.handle); err != nil {
		return fmt.Errorf("addinhost: closing %q: %w", l.path, err)
	}
	return nil
}

func (l *sharedLib) id() string       { return C.GoString(C.call_str(l.idSym)) }
func (l *sharedLib) manifest() string { return C.GoString(C.call_str(l.manSym)) }

// apiVersion reports the oblikovati.org/api major/minor the add-in was compiled
// against (ObkAddInApiMajor/ObkAddInApiMinor). present is false when the add-in omits
// either export, which the loader's gate treats as incompatible.
func (l *sharedLib) apiVersion() (major, minor int, present bool) {
	if l.majorSym == nil || l.minorSym == nil {
		return 0, 0, false
	}
	return int(C.call_int(l.majorSym)), int(C.call_int(l.minorSym)), true
}

func (l *sharedLib) activate() error {
	if rc := C.call_activate(l.actSym); rc != C.int(C.OBK_OK) {
		return fmt.Errorf("addinhost: ObkAddInActivate %q returned %d", l.path, int(rc))
	}
	return nil
}

func (l *sharedLib) deactivate() error {
	if rc := C.call_void(l.deactSym); rc != C.int(C.OBK_OK) {
		return fmt.Errorf("addinhost: ObkAddInDeactivate %q returned %d", l.path, int(rc))
	}
	return nil
}

func (l *sharedLib) notify(b []byte) error {
	var p *C.uint8_t
	if len(b) > 0 {
		p = (*C.uint8_t)(unsafe.Pointer(&b[0]))
	}
	if rc := C.call_notify(l.notifSym, p, C.int(len(b))); rc != C.int(C.OBK_OK) {
		return fmt.Errorf("addinhost: ObkAddInNotify %q returned %d", l.path, int(rc))
	}
	return nil
}

func (l *sharedLib) hasAutomation() bool { return l.autoSym != nil }

// automation invokes the add-in's optional ObkAddInAutomation export. The reply
// buffer is allocated by the add-in's runtime, so it is copied out and released
// through the add-in's own ObkFree (the cross-runtime ownership rule of the header).
func (l *sharedLib) automation(method string, req []byte) ([]byte, error) {
	if l.autoSym == nil {
		return nil, fmt.Errorf("addinhost: add-in %q exports no ObkAddInAutomation", l.path)
	}
	cm := C.CString(method)
	defer C.free(unsafe.Pointer(cm))
	var reqPtr *C.uint8_t
	if len(req) > 0 {
		reqPtr = (*C.uint8_t)(unsafe.Pointer(&req[0]))
	}
	var resp *C.uint8_t
	var respLen C.int
	rc := C.call_automation(l.autoSym, cm, reqPtr, C.int(len(req)), &resp, &respLen) //nolint:gocritic // dupSubExpr false positive from cgo's generated call site
	var out []byte
	if resp != nil {
		out = C.GoBytes(unsafe.Pointer(resp), respLen)
		C.call_addin_free(l.freeSym, resp)
	}
	if rc != C.int(C.OBK_OK) {
		return nil, fmt.Errorf("addinhost: automation %q on %q failed: %s", method, l.path, string(out))
	}
	return out, nil
}
