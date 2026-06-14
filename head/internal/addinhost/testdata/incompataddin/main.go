// SPDX-License-Identifier: GPL-2.0-only

// incompatfixture is a minimal c-shared add-in whose ObkAddInApiMajor reports a major
// (99) deliberately different from any real host's, so the loader's compatibility gate
// must refuse to load it. It implements just enough of the C ABI to be a loadable
// library; it is never activated.
package main

/*
#include <stdlib.h>
#include <stdint.h>

#define OBK_OK 0

typedef int  (*HostCall)(const char* method, const uint8_t* req, int reqLen, uint8_t** resp, int* respLen);
typedef void (*HostFree)(uint8_t* p);
*/
import "C"

var (
	idStr  = C.CString("com.oblikovati.incompat-fixture")
	manStr = C.CString(`{"id":"com.oblikovati.incompat-fixture","displayName":"Incompat Fixture","version":"0.0.0"}`)
)

//export ObkAddInId
func ObkAddInId() *C.char { return idStr }

//export ObkAddInManifest
func ObkAddInManifest() *C.char { return manStr }

// ObkAddInApiMajor reports an impossible major so the host's gate refuses the add-in;
// ObkAddInApiMinor is present (both version exports required) but irrelevant here.
//
//export ObkAddInApiMajor
func ObkAddInApiMajor() C.int { return 99 }

//export ObkAddInApiMinor
func ObkAddInApiMinor() C.int { return 0 }

//export ObkAddInActivate
func ObkAddInActivate(call C.HostCall, free C.HostFree) C.int { return C.OBK_OK }

//export ObkAddInDeactivate
func ObkAddInDeactivate() C.int { return C.OBK_OK }

//export ObkAddInNotify
func ObkAddInNotify(ev *C.uint8_t, n C.int) C.int { return C.OBK_OK }

//export ObkFree
func ObkFree(p *C.uint8_t) {}

func main() {}
