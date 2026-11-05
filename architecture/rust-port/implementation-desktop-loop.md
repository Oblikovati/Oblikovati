# Desktop executable and native owner loop — partial R1V

Tracking: [#3630](https://github.com/Oblikovati/Oblikovati/issues/3630), under
[ADR-0008](../decisions/ADR-0008-vulkan-renderer.md). This is implementation
evidence; the accepted ADR and the milestone exit remain unchanged.

## Source and current behavior

`crates/head` supplies the `oblikovati-head` binary and its desktop composition
root. The fixed Go reference is
`aa8297d1b0ac33c024f19e54821536d76e8a447a`,
`head/cmd/oblikovati-head/main.go`. The Go head's `-frames` smoke mode and #1493
idle event wait supply the lifecycle requirements for this step.

The Rust head opens one application-owned X11 viewport through the released
Martensite adapter. It starts with an empty presentation. It does not seed fake
geometry or report a completed CAD workspace. With no frame bound it renders on
startup, exposure, resize, focus change, remap, or an explicit worker wake. It
waits on native input and the coalesced wake signal when idle. A hidden window
waits for remap and cannot report offscreen presentation as a native frame.

`--frames N` and `-frames N` bound presentation attempts; zero selects the normal
window lifetime. A bounded run makes no idle event waits. It refuses a hidden
window and requires at least one completed native frame before reporting success.
A measured surface change schedules the next frame with the new surface input.
It does not count as completed. A renderer failure ends the run with an error.
The view retires on success, close, or an input/render/wait failure. A cleanup
failure is preserved even when another error has already occurred.

The head's `--help` and `--version` need no display. Version output consumes the
same checked build identity as the CLI. Publication stays disabled in
`release.toml`; adding the binary does not certify the application or package.
Windows and macOS pipelines remain unchanged and deferred.

## Local acceptance

```sh
cargo test --locked -p oblikovati-head -p oblikovati-viewport
cargo run --locked -p oblikovati-head -- --frames 3
make gate
```

Named fake windows check bounded polling, idle wake behavior, hide/remap, early
close, surface changes, failed I/O, frame-count exhaustion, and cleanup errors.
They verify loop decisions, not rendering. The native tests run the actual Linux
binary and library composition root. They require completed frames and empty
Vulkan validation errors. Another native test uses a model-free worker wake,
hides and remaps the real window, and retires the view. The viewport's separate
live test still compares actual native pixels with GPU captures.

All checks remain on this Linux host. No remote push, CI test, or other platform
run forms part of this evidence.

## Remaining integration

The executable currently shows a clear viewport. The
[desktop document owner](implementation-desktop-owner.md) now binds document
ownership and typed worker requests to this loop. Native add-in registry and C
callback attachment, saved window state, icons, and model presentations
remain open. Shared Dear ImGui composition requires the
kernel-owned interface in [viewport evidence](implementation-viewport.md).
Picking, UI controls, and foreground MCPBridge acceptance remain required.
This step does not establish a usable CAD desktop or complete R1V/R4/R7.

The [desktop script composition](implementation-desktop-scripts.md) now uses timed
owner waits while an interpreter is active. Empty polls do not force a GPU
redraw. Native script/editor widgets remain open work.
