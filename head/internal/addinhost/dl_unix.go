//go:build linux || darwin

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// openLibrary dlopens path and resolves the required ObkAddIn* exports into a
// sharedLib (dl_shared.go holds the portable export-invocation code; this file holds
// only the dlfcn loader primitives).
func openLibrary(path string) (addInLib, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	h := C.dlopen(cpath, C.RTLD_NOW|C.RTLD_LOCAL)
	if h == nil {
		return nil, fmt.Errorf("addinhost: dlopen %q: %s", path, C.GoString(C.dlerror()))
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

// close unmaps the shared library (dlclose); a non-zero return is an error.
func (l *sharedLib) close() error {
	if rc := C.dlclose(l.handle); rc != 0 {
		return fmt.Errorf("addinhost: dlclose %q: %s", l.path, C.GoString(C.dlerror()))
	}
	return nil
}
