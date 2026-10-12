# Native add-in desktop services — partial R4

`NativeAddInDesktop` implements the production `HeadWork` boundary and borrows the
existing `ScriptDocumentOwner`. It installs the existing native callback runtime
with an injected wake signal. Foreign library calls stay on that runtime's single
worker; it owns no document or window. The model is borrowed only while the UI
owner drains copied callback requests. Constructor failure does not consume the
document owner.

Load and activation are asynchronous controls with explicit unsafe trust and
lifetime requirements. Each completion must be consumed before another control
starts. A worker failure is retained as an error result. Polling divides a bounded
request budget between native callbacks and existing owner services. In-flight
foreign work and ready callback requests are separate: ready work prevents sleep;
a busy worker wakes the native window when a callback or completion arrives.
Inner services must also accept their allocated budget; default frames supply 32.

Scene publication, GUI metadata, Save path choices, native command inputs, command
feedback, and script host access delegate to the same document owner. This is
service composition, not another model, history, file codec, or API DTO set.
The native catalog continues to wrap the native callback transport. Complete
lifecycle wire routing for other transports remains open.

Close is refused while a native control is pending, then uses the existing owner
close guard. Explicit finish drains pending foreign work, attempts registry
shutdown, retires callbacks, joins the worker, and checks reply ownership before
retiring the document owner. A failed foreign cleanup leaves the owner available
for retry. A prior control failure is reported even when cleanup succeeds.
Repeated successful finish does not restart native work. Drop remains insufficient
for foreign cleanup; native libraries remain mapped under the existing policy.
A trusted in-process library that hangs cannot be forcibly cancelled safely.

## Local evidence

A process-isolated native-window test opens the real Linux head before starting
foreign work, preserving Xlib initialization order. Its injected startup controller
loads real API-only Rust and Go document add-ins, activates each, and services their
callbacks through the production frame/wait loop. They create constrained sketches,
edit parameters, Undo/Redo, save, close, and reopen real native files. The owner
independently checks both reopened sketches, radius against the authored 3.5
centimetres at the configured solver precision, identical radius bits between
languages, the
width parameter, and clean state. Shutdown runs while the owner is still available.

A second case begins a missing-library load and immediately finishes. The original
load failure is returned after native cleanup, and the borrowed owner remains
available. It also checks GUI and Save delegation, the canonical script host, and
retired-control refusal. The frame test checks budget rejection and stopped polling.
The shared fixture compiler takes an explicit source root so both consumers use
the same Go/Rust corpus; it does not duplicate the workflow source.

The tests run with software Vulkan on this Linux host, with validation enabled.
They prove native frame service and typed model/file operations. They do not prove
solid feature rendering or foreground MCPBridge behavior. Executable startup
selection, installed-bundle discovery, policy persistence, native event delivery,
full lifecycle wire routing, and MCPBridge acceptance remain open. The executable's
default startup does not yet construct this service adapter. No kernel source,
release pin, deferred-platform test, remote push, or PR is part of this work.

Full local `make gate` passed with software Vulkan and validation enabled across
all thirteen crates. Production line coverage was 95.12% for head and 96.75% for
the native add-in host; duplication remained below 3%. The known AMD native
presentation issue remains a separate open acceptance item.
