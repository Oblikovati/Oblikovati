// SPDX-License-Identifier: GPL-2.0-only

// echofixture is a minimal c-shared add-in used by the addinhost loader integration
// test. On Activate it calls the host's echo callback with "ping" and returns OBK_OK
// iff it gets "ping" back — exercising the whole C-ABI seam (host->add-in call,
// add-in->host call, host-owned buffer read + free) across two Go runtimes.
package main

/*
#include <stdlib.h>
#include <stdint.h>

#define OBK_OK 0
#define OBK_ERR 1

typedef int  (*HostCall)(const char* method, const uint8_t* req, int reqLen, uint8_t** resp, int* respLen);
typedef void (*HostFree)(uint8_t* p);

static int  call_host(HostCall c, const char* m, const uint8_t* req, int n, uint8_t** resp, int* rl) { return c(m, req, n, resp, rl); }
static void call_free(HostFree f, uint8_t* p) { f(p); }
*/
import "C"
import "unsafe"

var (
	idStr  = C.CString("com.oblikovati.echo-fixture")
	manStr = C.CString(`{"id":"com.oblikovati.echo-fixture","displayName":"Echo Fixture","version":"0.0.0","capabilities":[]}`)
)

//export ObkAddInId
func ObkAddInId() *C.char { return idStr }

//export ObkAddInManifest
func ObkAddInManifest() *C.char { return manStr }

//export ObkAddInActivate
func ObkAddInActivate(call C.HostCall, free C.HostFree) C.int {
	method := C.CString("echo")
	defer C.free(unsafe.Pointer(method))
	req := []byte("ping")
	var resp *C.uint8_t
	var respLen C.int
	rc := C.call_host(call, method, (*C.uint8_t)(unsafe.Pointer(&req[0])), C.int(len(req)), &resp, &respLen)
	if rc != C.OBK_OK || resp == nil {
		return C.OBK_ERR
	}
	got := C.GoBytes(unsafe.Pointer(resp), respLen)
	C.call_free(free, resp)
	if string(got) == "ping" {
		return C.OBK_OK
	}
	return C.OBK_ERR
}

//export ObkAddInDeactivate
func ObkAddInDeactivate() C.int { return C.OBK_OK }

//export ObkAddInNotify
func ObkAddInNotify(ev *C.uint8_t, n C.int) C.int { return C.OBK_OK }

//export ObkFree
func ObkFree(p *C.uint8_t) { C.free(unsafe.Pointer(p)) }

// ObkAddInAutomation is the OPTIONAL automation export (M05-F01 #252): it echoes
// the method and payload back as JSON so the host-side automation round-trip test
// can assert the buffer crossed both runtimes intact. Allocated with C.CBytes so
// the host's release through ObkFree (C.free) matches.
//
//export ObkAddInAutomation
func ObkAddInAutomation(method *C.char, req *C.uint8_t, n C.int, resp **C.uint8_t, respLen *C.int) C.int {
	var payload []byte
	if req != nil {
		payload = C.GoBytes(unsafe.Pointer(req), n)
	}
	out := []byte(`{"method":"` + C.GoString(method) + `","payload":"` + string(payload) + `"}`)
	*resp = (*C.uint8_t)(C.CBytes(out))
	*respLen = C.int(len(out))
	return C.OBK_OK
}

func main() {}
