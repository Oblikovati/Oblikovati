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
typedef int  (*activateFn)(ObkHostCall, ObkHostFree);
typedef int  (*voidFn)(void);
typedef int  (*notifyFn)(const uint8_t*, int);

static const char* call_str(void* fn)               { return ((strFn)fn)(); }
static int  call_activate(void* fn)                  { return ((activateFn)fn)((ObkHostCall)ObkHostDispatch, (ObkHostFree)ObkHostReleaseBuf); }
static int  call_void(void* fn)                      { return ((voidFn)fn)(); }
static int  call_notify(void* fn, const uint8_t* ev, int n) { return ((notifyFn)fn)(ev, n); }
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// unixLib is a dlopen'd add-in with its resolved C entry points.
type unixLib struct {
	handle   unsafe.Pointer
	idSym    unsafe.Pointer
	manSym   unsafe.Pointer
	actSym   unsafe.Pointer
	deactSym unsafe.Pointer
	notifSym unsafe.Pointer
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
	}
	for _, s := range syms {
		p, err := resolve(h, s.name)
		if err != nil {
			C.dlclose(h)
			return nil, err
		}
		*s.dst = p
	}
	return l, nil
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

func (l *unixLib) close() error {
	if rc := C.dlclose(l.handle); rc != 0 {
		return fmt.Errorf("addinhost: dlclose %q: %s", l.path, C.GoString(C.dlerror()))
	}
	return nil
}
