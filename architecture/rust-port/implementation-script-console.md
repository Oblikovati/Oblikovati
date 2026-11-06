# Script Console controller and frame state — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This ports Go `script/console/controller.go` and `console.go` onto the
[isolated Lua execution interface](implementation-cli-lua.md), under ADR-0009/0011.
It provides the controller and copied state for a native panel; it does not create
or claim a rendered panel, editor or foreground UI acceptance.

`ScriptConsoleController` owns one engine, execution limits and a stable deduplicated
host method inventory. Run and Check return after starting a worker process.
The owner calls `poll` per frame, supplying its borrowed canonical host. Each poll
handles at most one stream event. Lua evaluation stays outside the application
process; the controller captures no document, GUID source or model caller.
A non-Send marker keeps controller/model dispatch on the same owner. The compile-fail
example proves that moving the controller to another thread is rejected.

Run and Check share the engine's execution lease; neither can start while the other
is active. Check uses the exact pinned Lua grammar in parse-only mode and makes no
host calls, prints or source execution. Its result is separate from the last Run
outcome. Stop cancels either mode and is harmless while idle. A stopped run remains
active until a frame records completion. Run errors retain earlier print records
and the failed outcome. Failed worker startup does not falsely mark a run active.

Snapshots copy output and results for one frame. Clear removes retained output and
its truncation count while leaving execution and last-result state unchanged,
matching the Go console behavior. Each print invocation is one record, including
embedded newlines. Oldest records drop after the Go limit of 5,000 records; a new
4 MiB retention bound also limits accumulated output across separate runs. The
snapshot exposes the number of dropped records, so retention loss is visible.
The worker's per-run output limit remains separate from console retention.

GUI limits retain the Go baseline's five-second wall bound and 128 MiB Lua memory
budget, with the Rust interpreter's additional positive instruction, stack, output,
host-call and source bounds. They do not turn model-call cancellation into rollback
or make a synchronous long modeling operation bounded by the VM's deadline.

Tests use actual isolated Lua workers and a named non-Send host. They check streamed
output and owner calls, native frame progress while Lua loops, busy refusal, Stop
and rerun, compile-only checks, separate outcomes, output copies, Clear while
running, partial output on failure and failed startup. Unit cases verify both
retention limits, visible drop counts, multiline record boundaries and copied views.

Run `cargo test -p oblikovati-script` for the controller, VM, process and compile-fail
checks, and `make gate` for full local Linux acceptance. The pinned Go console tests
remain the behavior oracle. Native panel bindings, editor buffer/input, completion
and help, document rules and responsive foreground MCPBridge tests remain open.
Windows and macOS remain deferred.

The [editor source core](implementation-script-editor.md) now owns Unicode source,
selection, reversible edits and basic caret movement. Remaining editor operations
and native input/render bindings are still open.
