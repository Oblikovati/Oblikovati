/* SPDX-License-Identifier: Apache-2.0 */
/*
 * oblikovati_addin.h — the C ABI between the Oblikovati host and a shared-library
 * add-in (built with `go build -buildmode=c-shared`, producing .so/.dll).
 *
 * This header is part of the public, Apache-2.0 automation contract: it is the
 * single source of truth for the host<->add-in boundary. Add-ins compile against
 * it; the host vendors a copy of it. Keep it self-contained (C primitives only).
 *
 * WHY a C ABI and not Go types: a Go c-shared library loaded into the Go host runs
 * its OWN Go runtime (two heaps, two GCs in one process). Go pointers must not cross
 * the boundary, so everything here is C primitives + byte buffers. The boundary is
 * therefore in-process RPC: the add-in sends a serialized (JSON, v1) request through
 * ObkHostCall; the host runs it against the live model on the session goroutine and
 * returns a serialized result. The JSON `method` strings are the API surface
 * (commands.*, documents.*, parameters.*, model.*, features.*).
 *
 * Memory ownership (the cross-runtime free footgun — be explicit):
 *   - A *resp buffer returned by ObkHostCall is owned by the HOST; the add-in must
 *     release it with the ObkHostFree passed to ObkAddInActivate.
 *   - Buffers the add-in returns to the host (none today; reserved) are freed by
 *     the add-in's ObkFree.
 *   - const char* returned by ObkAddInId/ObkAddInManifest are owned by the add-in
 *     and remain valid until ObkAddInDeactivate; the host must not free them.
 */
#ifndef OBLIKOVATI_ADDIN_H
#define OBLIKOVATI_ADDIN_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Status codes shared by host and add-in entry points. */
#define OBK_OK 0
#define OBK_ERR 1

/*
 * Host -> add-in: the single generic RPC entry. Runs `method` with the `req`
 * byte payload (reqLen bytes) and, on OBK_OK, sets *resp/*respLen to a
 * host-owned buffer the add-in must release with ObkHostFree. On error returns
 * OBK_ERR and *resp points to a host-owned UTF-8 error message (respLen bytes).
 */
typedef int (*ObkHostCall)(const char *method,
                           const uint8_t *req, int reqLen,
                           uint8_t **resp, int *respLen);

/* Host -> add-in: release a buffer handed back by ObkHostCall. */
typedef void (*ObkHostFree)(uint8_t *p);

/*
 * Exports every add-in MUST provide (resolved by the host after dlopen/LoadLibrary).
 *
 *   ObkAddInId        stable add-in id (e.g. "com.oblikovati.mcp-bridge").
 *   ObkAddInManifest  JSON: {id, displayName, version, capabilities:[...]}.
 *   ObkAddInActivate  store the host callbacks and start the add-in (e.g. its MCP
 *                     server). Returns OBK_OK on success.
 *   ObkAddInDeactivate stop the add-in and join its goroutines. Returns OBK_OK.
 *   ObkAddInNotify    host pushes a serialized event (e.g. DocumentCreated) for the
 *                     add-in to fan out (e.g. over SSE). Returns OBK_OK.
 *   ObkFree           release a buffer the add-in previously returned to the host.
 */
/*
 * These prototypes are for the HOST, which resolves them by name after loading the
 * library. The add-in itself *defines* them (via cgo //export, which emits its own
 * non-const prototypes), so it builds with -DOBK_BUILDING_ADDIN to skip these and
 * avoid a const-qualifier conflict. Both sides still share the typedefs/macros above.
 */
#ifndef OBK_BUILDING_ADDIN
extern const char *ObkAddInId(void);
extern const char *ObkAddInManifest(void);
extern int ObkAddInActivate(ObkHostCall call, ObkHostFree freeFn);
extern int ObkAddInDeactivate(void);
extern int ObkAddInNotify(const uint8_t *ev, int len);
extern void ObkFree(uint8_t *p);

/*
 * OPTIONAL export: the add-in's automation surface (ApplicationAddIn.Automation,
 * M05-F01 #252). The host resolves it leniently after load — an add-in without it
 * simply reports hasAutomation:false in the registry. When present, the host routes
 * addins.callAutomation requests here: `method` plus a JSON `req` chosen by THIS
 * add-in's own contract (the host passes both through opaquely). On OBK_OK the
 * add-in sets *resp/*respLen to a buffer it allocated, which the host copies and
 * then releases via ObkFree; on OBK_ERR *resp is a UTF-8 error message (same
 * ownership). Constraints: the call arrives on the host's session goroutine, so the
 * handler must return promptly and must NOT call ObkHostCall synchronously — the
 * dispatcher that would run it is the one waiting on this very call.
 */
extern int ObkAddInAutomation(const char *method,
                              const uint8_t *req, int reqLen,
                              uint8_t **resp, int *respLen);
#endif /* OBK_BUILDING_ADDIN */

#ifdef __cplusplus
}
#endif

#endif /* OBLIKOVATI_ADDIN_H */
