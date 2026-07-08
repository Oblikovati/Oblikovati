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
