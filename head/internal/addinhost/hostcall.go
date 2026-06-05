// SPDX-License-Identifier: GPL-2.0-only

// Package addinhost loads shared-library (.so/.dll) add-ins and bridges the C ABI
// (include/oblikovati_addin.h — the host's vendored copy of the ABI contract, kept
// byte-identical to the add-in repo's copy) to the live session. Add-ins run in their own
// Go runtime and reach the host only through a single C function pointer
// (ObkHostDispatch): they send a JSON request, the host runs it on the session
// goroutine via the Dispatcher, and returns a JSON result. See the package memory /
// plan for why the boundary is data-marshaling, not object-sharing (two Go runtimes).
package addinhost

/*
#cgo CFLAGS: -I${SRCDIR}/include
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include "oblikovati_addin.h"
*/
import "C"

import (
	"context"
	"sync"
	"time"
	"unsafe"

	"github.com/Oblikovati/oblikovati/addin/dispatch"
)

// Handler runs a host API method given its JSON request, returning a JSON result.
// It executes on the session goroutine (inside Dispatcher.Drain), so it may touch
// the non-thread-safe model.
type Handler func(method string, req []byte) ([]byte, error)

var (
	hostMu      sync.RWMutex
	hostDisp    *dispatch.Dispatcher
	hostHandler Handler
	hostTimeout time.Duration
)

// SetHost installs the dispatcher + handler that the C-ABI entry point routes
// through, and the per-call timeout. Call once before activating any add-in.
func SetHost(d *dispatch.Dispatcher, h Handler, timeout time.Duration) {
	hostMu.Lock()
	defer hostMu.Unlock()
	hostDisp, hostHandler, hostTimeout = d, h, timeout
}

//export ObkHostDispatch
func ObkHostDispatch(method *C.char, req *C.uint8_t, reqLen C.int, resp **C.uint8_t, respLen *C.int) C.int {
	hostMu.RLock()
	d, h, to := hostDisp, hostHandler, hostTimeout
	hostMu.RUnlock()
	if d == nil || h == nil {
		return writeErr(resp, respLen, "addinhost: host not initialized")
	}
	m := C.GoString(method)
	b := C.GoBytes(unsafe.Pointer(req), reqLen) // copy before Submit: req is add-in memory
	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()
	out, err := d.Submit(ctx, func() ([]byte, error) { return h(m, b) })
	if err != nil {
		return writeErr(resp, respLen, err.Error())
	}
	writeBuf(resp, respLen, out)
	return C.int(C.OBK_OK)
}

//export ObkHostReleaseBuf
func ObkHostReleaseBuf(p *C.uint8_t) { C.free(unsafe.Pointer(p)) }

// writeBuf hands the add-in a freshly malloc'd (host-owned) copy of b; the add-in
// frees it via ObkHostReleaseBuf. Always allocates so ownership is uniform.
func writeBuf(resp **C.uint8_t, respLen *C.int, b []byte) {
	buf := C.malloc(C.size_t(len(b)) + 1)
	if len(b) > 0 {
		C.memcpy(buf, unsafe.Pointer(&b[0]), C.size_t(len(b)))
	}
	*resp = (*C.uint8_t)(buf)
	*respLen = C.int(len(b))
}

func writeErr(resp **C.uint8_t, respLen *C.int, msg string) C.int {
	writeBuf(resp, respLen, []byte(msg))
	return C.int(C.OBK_ERR)
}
