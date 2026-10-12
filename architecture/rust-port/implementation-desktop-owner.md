# Desktop document owner and frame dispatch — partial R4

Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563) and
[#3630](https://github.com/Oblikovati/Oblikovati/issues/3630). The ownership rules
remain [ADR-0002](../decisions/ADR-0002-api-and-addin-host.md).
This is implementation evidence, not a new or edited accepted ADR.

## Composition and request lifetime

The native head now owns a `FileWorkspace`, GUID source, document observer
factory, and bounded API admission queue through `DesktopOwner`. It borrows the
canonical `FileHost` for each frame batch. Worker transports retain copied
requests and a wake signal, never documents, file access, native windows, or
GUID generation. UI commands can borrow `document_host` directly on the owner;
they cannot use synchronous worker admission from the owner thread. Frame polling
consumes observer invalidation even when the worker queue is empty.

The fixed Go source is `head/cmd/oblikovati-head/addins.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The default admission capacity remains
64, host-call timeout remains 10 seconds, and the loop drains at most 32 calls
per frame. Work left after that batch prevents an idle wait, including while the
window is hidden. A newly admitted call wakes an already idle X11 loop. Accepted
request timeout semantics are inherited from the existing dispatcher; there is
no automatic edit retry.

The standalone viewport test entry point and the document-backed native head use
one frame loop. Queue admission stops before window retirement. Pending callers
receive the canonical closed-dispatch error. Queue retirement does not close or
discard documents. `into_documents` returns ownership for explicit recovery or a
new window after a failed run. Foreign-library shutdown is not claimed by this
queue-only composition; library activation is not enabled here.

## Close and observations

A native close request follows the current frame's request drain. The owner
refuses it when requests remain queued or any open document has no save path,
has unsaved changes, or has an open transaction. Refused document IDs remain
available for a future save prompt. The window stays alive; a later close request
runs the check again. A positive smoke frame bound is an explicit test exit,
not an implicit document discard operation.

Desktop observations coalesce invalidation and retain unavailable-interest-client
warnings by document. Closing a document removes its warning. The foreign
registry and UI notification panel remain unbound, so this observer does not
claim that a registered add-in is present. It preserves the warning for later
consumption instead of dropping it.

## Local Linux acceptance

```sh
cargo test --locked -p oblikovati-head
make gate
```

Typed worker tests create, save, start and abort a transaction, close, and reopen
through the real API client and canonical file owner. Named memory files and an
owner-thread GUID source make these orchestration tests independent of external
I/O. Tests also cover a refused native close, remaining queue batches, rejected
owner-thread admission, pending-waiter release, and ownership recovery.

The native window test executes the same production frame/owner composition on
this Linux host. A typed worker creates a part, saves a real native package,
closes it, and reopens it while native frames continue. It posts a completion
wake and an injected close request only after the workflow finishes. The owner
independently checks the reopened document identity, clean state, save path, and
completed native frames. The injected close is a loop test input, not proof of a
window-manager close message; native protocol translation has separate tests.

## Remaining work

The executable still shows a clear viewport. Model presentations, shared Dear
ImGui composition, UI commands and save prompts, scripts, native add-in registry
attachment and C callbacks, event subscriptions, and foreground MCPBridge tests
remain open. This work does not establish a usable CAD desktop or complete
R1V/R4/R7. Tests remain local to Linux; no remote push or CI run is included.
