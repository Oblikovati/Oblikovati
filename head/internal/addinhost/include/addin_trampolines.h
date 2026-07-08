/* SPDX-License-Identifier: GPL-2.0-only */
/*
 * addin_trampolines.h — the portable C shared by the Unix (dl_unix.go) and Windows
 * (dl_windows.go) add-in loaders. It holds everything that is NOT the OS-specific
 * loader primitive (dlopen vs LoadLibrary, dlsym vs GetProcAddress, dlclose vs
 * FreeLibrary): the host's C-ABI callback declarations, the function-pointer
 * typedefs for the add-in's exports, and the call_* trampolines.
 *
 * WHY trampolines: cgo cannot call a Go-held C function pointer directly, so each
 * platform resolves a symbol to a void* (dlsym / GetProcAddress) and invokes it here
 * in C by casting to its typedef. The invoking code is then identical on both OSes
 * (dl_shared.go), and only the ~3 loader primitives differ per platform — no
 * duplication of the trampoline C (repo CLAUDE.md "no code duplication").
 */
#ifndef OBLIKOVATI_ADDIN_TRAMPOLINES_H
#define OBLIKOVATI_ADDIN_TRAMPOLINES_H

#include <stdint.h>
#include <stdlib.h>
#include "oblikovati_addin.h"

/* The host's C-ABI callbacks (defined via //export in hostcall.go). Declared with
 * non-const params to match cgo's generated prototypes; the casts below adapt them
 * to the (const-qualified) ObkHostCall/ObkHostFree typedefs. */
extern int  ObkHostDispatch(char* method, uint8_t* req, int reqLen, uint8_t** resp, int* respLen);
extern void ObkHostReleaseBuf(uint8_t* p);

/* Function-pointer typedefs for the add-in's ObkAddIn* exports. */
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

#endif /* OBLIKOVATI_ADDIN_TRAMPOLINES_H */
