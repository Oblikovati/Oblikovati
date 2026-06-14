//go:build linux || darwin

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

/*
#cgo CFLAGS: -I${SRCDIR}/include
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <stdint.h>
#include "oblikovati_addin.h"

// The host's C-ABI callbacks (defined via //export in hostcall.go). Declared with
// non-const params to match cgo's generated prototypes; the casts below adapt them
// to the (const-qualified) ObkHostCall/ObkHostFree typedefs.
extern int  ObkHostDispatch(char* method, uint8_t* req, int reqLen, uint8_t** resp, int* respLen);
extern void ObkHostReleaseBuf(uint8_t* p);

// Function-pointer trampolines: cgo cannot call a Go-held C pointer directly, so we
// cast each dlsym result to its typedef and invoke it from C.
typedef const char* (*strFn)(void);
typedef int  (*intFn)(void);
typedef int  (*activateFn)(ObkHostCall, ObkHostFree);
typedef int  (*voidFn)(void);
typedef int  (*notifyFn)(const uint8_t*, int);
typedef int  (*automationFn)(const char*, const uint8_t*, int, uint8_t**, int*);
typedef void (*freeFn)(uint8_t*);

static const char* call_str(void* fn)               { return ((strFn)fn)(); }
static int  call_int(void* fn)                       { return ((intFn)fn)(); }
static int  call_activate(void* fn)                  { return ((activateFn)fn)((ObkHostCall)ObkHostDispatch, (ObkHostFree)ObkHostReleaseBuf); }
static int  call_void(void* fn)                      { return ((voidFn)fn)(); }
static int  call_notify(void* fn, const uint8_t* ev, int n) { return ((notifyFn)fn)(ev, n); }
static int  call_automation(void* fn, const char* method, const uint8_t* req, int n, uint8_t** resp, int* respLen) { return ((automationFn)fn)(method, req, n, resp, respLen); }
static void call_addin_free(void* fn, uint8_t* p)    { ((freeFn)fn)(p); }
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// unixLib is a dlopen'd add-in with its resolved C entry points. autoSym is the
// OPTIONAL ObkAddInAutomation export (nil when the add-in has no automation
// surface, M05-F01 #252); freeSym is the add-in's ObkFree, which releases the
// buffers automation hands back across the runtime boundary.
type unixLib struct {
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

// openLibrary dlopens path and resolves the required ObkAddIn* exports.
func openLibrary(path string) (addInLib, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	h := C.dlopen(cpath, C.RTLD_NOW|C.RTLD_LOCAL)
	if h == nil {
		return nil, fmt.Errorf("addinhost: dlopen %q: %s", path, C.GoString(C.dlerror()))
	}
	l := &unixLib{handle: h, path: path}
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
			C.dlclose(h)
			return nil, err
		}
		*s.dst = p
	}
	// Automation is an optional export: absence just means hasAutomation() is false.
	l.autoSym = resolveOptional(h, "ObkAddInAutomation")
	// The version exports are resolved leniently so a missing one does not abort the
	// whole load; the compatibility gate (loader.go) turns their absence into a clear
	// "cannot verify compatibility" skip rather than a dlsym error.
	l.majorSym = resolveOptional(h, "ObkAddInApiMajor")
	l.minorSym = resolveOptional(h, "ObkAddInApiMinor")
	return l, nil
}

// resolveOptional looks up a symbol the contract marks optional; nil means absent.
func resolveOptional(h unsafe.Pointer, name string) unsafe.Pointer {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	return C.dlsym(h, cn)
}

// resolve looks up a required symbol, returning a descriptive error if absent.
func resolve(h unsafe.Pointer, name string) (unsafe.Pointer, error) {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	p := C.dlsym(h, cn)
	if p == nil {
		return nil, fmt.Errorf("addinhost: dlsym %q: %s", name, C.GoString(C.dlerror()))
	}
	return p, nil
}

func (l *unixLib) id() string       { return C.GoString(C.call_str(l.idSym)) }
func (l *unixLib) manifest() string { return C.GoString(C.call_str(l.manSym)) }

// apiVersion reports the oblikovati.org/api major/minor the add-in was compiled
// against (ObkAddInApiMajor/ObkAddInApiMinor). present is false when the add-in omits
// either export, which the loader's gate treats as incompatible.
func (l *unixLib) apiVersion() (major, minor int, present bool) {
	if l.majorSym == nil || l.minorSym == nil {
		return 0, 0, false
	}
	return int(C.call_int(l.majorSym)), int(C.call_int(l.minorSym)), true
}

func (l *unixLib) activate() error {
	if rc := C.call_activate(l.actSym); rc != C.int(C.OBK_OK) {
		return fmt.Errorf("addinhost: ObkAddInActivate %q returned %d", l.path, int(rc))
	}
	return nil
}

func (l *unixLib) deactivate() error {
	if rc := C.call_void(l.deactSym); rc != C.int(C.OBK_OK) {
		return fmt.Errorf("addinhost: ObkAddInDeactivate %q returned %d", l.path, int(rc))
	}
	return nil
}

func (l *unixLib) notify(b []byte) error {
	var p *C.uint8_t
	if len(b) > 0 {
		p = (*C.uint8_t)(unsafe.Pointer(&b[0]))
	}
	if rc := C.call_notify(l.notifSym, p, C.int(len(b))); rc != C.int(C.OBK_OK) {
		return fmt.Errorf("addinhost: ObkAddInNotify %q returned %d", l.path, int(rc))
	}
	return nil
}

func (l *unixLib) hasAutomation() bool { return l.autoSym != nil }

// automation invokes the add-in's optional ObkAddInAutomation export. The reply
// buffer is allocated by the add-in's runtime, so it is copied out and released
// through the add-in's own ObkFree (the cross-runtime ownership rule of the header).
func (l *unixLib) automation(method string, req []byte) ([]byte, error) {
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

func (l *unixLib) close() error {
	if rc := C.dlclose(l.handle); rc != 0 {
		return fmt.Errorf("addinhost: dlclose %q: %s", l.path, C.GoString(C.dlerror()))
	}
	return nil
}
