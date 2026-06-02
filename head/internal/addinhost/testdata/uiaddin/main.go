// SPDX-License-Identifier: GPL-2.0-only

// uifixture is a c-shared add-in for the addinhost UI-extension test. It demonstrates
// the Inventor-style flow end to end over the C ABI: on Activate it registers a ribbon
// BUTTON through the host API (commands.create); when the user clicks that button the
// host fires a command.ended event, delivered to ObkAddInNotify, where the add-in runs
// its action — here, creating a document through the host API (documents.create).
//
// The action runs on its OWN goroutine: ObkAddInNotify is invoked on the host's
// session goroutine, and a host call made synchronously from there would block on the
// dispatcher that very goroutine drives (a deadlock). Real add-ins relay events
// asynchronously for the same reason.
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

import (
	"encoding/json"
	"sync"
	"unsafe"
)

// buttonID is the command this add-in registers and listens for.
const buttonID = "AddIn.Ping"

var (
	idStr  = C.CString("com.oblikovati.ui-fixture")
	manStr = C.CString(`{"id":"com.oblikovati.ui-fixture","displayName":"UI Fixture","version":"0.0.0","capabilities":["ui"]}`)

	mu       sync.Mutex
	hostCall C.HostCall
	hostFree C.HostFree
)

//export ObkAddInId
func ObkAddInId() *C.char { return idStr }

//export ObkAddInManifest
func ObkAddInManifest() *C.char { return manStr }

//export ObkAddInActivate
func ObkAddInActivate(call C.HostCall, free C.HostFree) C.int {
	mu.Lock()
	hostCall, hostFree = call, free
	mu.Unlock()
	// Extend the UI: register a large-icon ribbon button (Inventor's ButtonDefinition).
	req, _ := json.Marshal(map[string]any{
		"id":          buttonID,
		"displayName": "Ping",
		"tab":         "AddInTab",
		"category":    "Demo",
		"icon":        "extrude",
		"buttonStyle": 2, // LargeIconButton
	})
	if _, ok := callHost("commands.create", req); !ok {
		return C.OBK_ERR
	}
	return C.OBK_OK
}

//export ObkAddInDeactivate
func ObkAddInDeactivate() C.int { return C.OBK_OK }

// hostEvent is the subset of the host's serialized event this add-in reacts to.
type hostEvent struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

//export ObkAddInNotify
func ObkAddInNotify(ev *C.uint8_t, n C.int) C.int {
	var e hostEvent
	if err := json.Unmarshal(C.GoBytes(unsafe.Pointer(ev), n), &e); err != nil {
		return C.OBK_OK // ignore events we cannot parse
	}
	if e.Type == "command.ended" && e.Command == buttonID {
		go runButtonAction() // off the session goroutine — see the package comment
	}
	return C.OBK_OK
}

// runButtonAction is the button's behavior: create a part document through the host
// API, proving an add-in can drive the model in response to its own button click.
func runButtonAction() {
	req, _ := json.Marshal(map[string]any{"type": "part", "name": "FromAddIn"})
	callHost("documents.create", req)
}

// callHost invokes a host method with a JSON request, returning the JSON reply and
// whether the call succeeded; it releases the host-owned reply buffer.
func callHost(method string, req []byte) ([]byte, bool) {
	mu.Lock()
	call, free := hostCall, hostFree
	mu.Unlock()
	if call == nil {
		return nil, false
	}
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))

	var reqPtr *C.uint8_t
	if len(req) > 0 {
		reqPtr = (*C.uint8_t)(unsafe.Pointer(&req[0]))
	}
	var resp *C.uint8_t
	var respLen C.int
	rc := C.call_host(call, cMethod, reqPtr, C.int(len(req)), &resp, &respLen)
	if resp == nil {
		return nil, rc == C.OBK_OK
	}
	out := C.GoBytes(unsafe.Pointer(resp), respLen)
	C.call_free(free, resp)
	return out, rc == C.OBK_OK
}

//export ObkFree
func ObkFree(p *C.uint8_t) { C.free(unsafe.Pointer(p)) }

func main() {}
