# Desktop isolated script execution — partial R7

Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563), under
[the API/host boundary](../decisions/ADR-0002-api-and-addin-host.md).
This extends [desktop ownership](implementation-desktop-owner.md) and
[console state](implementation-script-console.md). UI binding remains open.

## Parent and worker ownership

`DesktopScripts` joins the existing console controller to the document-backed
frame owner. The native head selects its own executable's private Lua worker
entry. That entry processes worker IPC before any native window or Vulkan
initialization. Lua runs only in the child. File access, GUID generation, model
calls, and transaction ownership remain in the parent through the same canonical
`FileHost` used by UI and queued clients. The console declares the implemented
file-host method inventory; it does not create a second wire definition.

Run and Check remain separate operations. Check compiles source and cannot run
model calls. Stop and shutdown retain committed edits. Shutdown cancels and
retires the isolated child before queue admission and the native view are
retired. An active run or check refuses window close; document dirtiness and open
transactions still apply after it completes. Returning the document owner first
retires any remaining interpreter job.

## Frame scheduling and invalidation

The console reports activity without cloning retained output each poll. Startup,
output, host calls, completion, Stop, and output clearing invalidate console
state. An active interpreter with no stream event does not itself request a GPU
redraw. It requests a timed owner wait at the existing 60 Hz animation cadence,
so the parent can service host calls and cancellation even when the child emits
nothing. Once idle, the console removes that timer. A shorter owner deadline is
preserved. Native input and queued client wakes still interrupt the wait.

One active script reserves one stream event from the frame's request budget;
the remaining budget services queued worker calls. The native loop therefore
services at most 31 queued calls plus one script event while a script is active,
and up to 32 queued calls otherwise. A budget too small for both sources is a
named decline before any calls run. Interpreter errors remain copied console
outcomes; they do not masquerade as renderer errors.

## Local Linux checks

```sh
cargo test --locked -p oblikovati-head -p oblikovati-script
make gate
```

Tests use the actual head worker entry with an invalid child DISPLAY to prove
that private execution does not initialize graphics. They exercise parent-owned
document creation, compile-only checks, active-child shutdown, cancellation after
a committed edit, and explicit ownership recovery. Separate console tests verify
empty active polls, output/host invalidation, retained output, and engine reuse.
A named fake window verifies that timed owner polling does not force a second
unchanged presentation.

The native test runs the same window/frame/document/script composition on this
Linux display. Lua creates a part, saves a real package, closes it, and reopens it
while native frames continue. The parent checks the clean reopened workspace,
file existence, output, and completed frames. Its injected close request stays
buffered until the test loop consumes it; script failures are reported before
that fixture attempts a close. This injected event is not window-manager close
acceptance. All testing remains on this host; no push or remote CI run is used.

## Remaining work

The head still shows a clear viewport. Dear ImGui console/editor widgets,
keyboard and clipboard binding, model presentation, native add-in registry and C
callback attachment, save prompts, event subscriptions, and foreground MCPBridge
acceptance remain required. This step does not complete R1V, R4, or R7.
