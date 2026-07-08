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

// openNativeLibrary dlopens path — the dlfcn loader primitive behind the shared
// openLibrary in dl_shared.go. RTLD_NOW surfaces missing symbols immediately;
// RTLD_LOCAL keeps the add-in's symbols out of the global namespace.
func openNativeLibrary(path string) (unsafe.Pointer, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	h := C.dlopen(cpath, C.RTLD_NOW|C.RTLD_LOCAL)
	if h == nil {
		return nil, fmt.Errorf("addinhost: dlopen %q: %s", path, C.GoString(C.dlerror()))
	}
	return h, nil
}

// lookupOptionalSymbol resolves a symbol the contract marks optional; nil means absent.
func lookupOptionalSymbol(h unsafe.Pointer, name string) unsafe.Pointer {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	return C.dlsym(h, cn)
}

// lookupSymbol resolves a required symbol, returning an error naming the symbol and the
// dlerror text if absent.
func lookupSymbol(h unsafe.Pointer, name string) (unsafe.Pointer, error) {
	if p := lookupOptionalSymbol(h, name); p != nil {
		return p, nil
	}
	return nil, fmt.Errorf("addinhost: dlsym %q: %s", name, C.GoString(C.dlerror()))
}

// closeNativeLibrary dlcloses the handle; a non-zero return is an error (the shared
// close() in dl_shared.go tags the add-in path onto it).
func closeNativeLibrary(h unsafe.Pointer) error {
	if rc := C.dlclose(h); rc != 0 {
		return fmt.Errorf("dlclose: %s", C.GoString(C.dlerror()))
	}
	return nil
}
