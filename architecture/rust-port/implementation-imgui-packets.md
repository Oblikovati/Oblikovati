# Dear ImGui owned draw packets — partial R1V/R4

`oblikovati-gui` wraps Dear ImGui 0.12.0 behind an owner-thread engine. A frame
returns owned positions, UVs, sRGB color bytes, indices, clip rectangles, and
selected Save/Undo/Redo/Home actions. The font atlas is owned independently.
The engine disables automatic ini/log file I/O and retains one active context
on the GUI owner thread until it closes. Invalid frame input and occupied context
creation are named errors.

The input supplies elapsed time, ordered input events, framebuffer extent, and
scale. Events carry physical pointer positions, left/middle/right button changes,
horizontal/vertical wheel amounts, named physical keys, aggregate modifiers, and
Unicode text. Logical coordinates must remain finite after scale conversion;
a positive scale alone cannot ensure this. Copying subtracts display origin and applies framebuffer
scale. Index offsets are checked before access; each command retains only the
vertices it uses. Texture IDs are checked against the font binding. Custom raw
callbacks are refused before a packet can be returned. Reset commands need no
mutable renderer state because each compiled draw specifies its complete state.
The [upstream draw-data contract](https://github.com/imgui-rs/imgui-rs/blob/v0.12.0/imgui/src/render/draw_data.rs)
limits borrowed data to the current frame; the application packets survive the
next frame and destruction of the context.

## Input ownership

Every event is validated before queue mutation. A refused
batch cannot leave an earlier click queued. Pointer positions are converted to
logical coordinates once; key identifiers are mapped by name inside the wrapper.
Aggregate Control/Shift/Alt/Super state is separate from the physical left/right
modifier keys. The native backend must provide both. Text is UTF-8; embedded NUL
is refused instead of passing a truncated string to the C interface.

`GuiFrameOutput` owns draws, selected actions, and pointer/keyboard/text capture
requests from that same frame. All events still go to Dear ImGui. The native
owner must use the capture requests and the Go command-chord rules to decide
which events also reach tools, global commands, and the viewport. This attachment
is not yet implemented. The engine preserves Dear ImGui's ordered event queue;
fast press/release pairs can span later frames without losing clicks. Native
frame scheduling must continue while queued input and active UI work remain.

The pinned binding uses its docking build and enables docking under ADR-0009.
Panel layout and native composition remain open. Tests use a real text widget to
check Unicode outside the basic multilingual plane, full-character deletion,
capture flags, pointer scaling, all three buttons, wheel values, and modifier
state. Two queued clicks survive four frames. An invalid batch leaves no click
behind. Automatic clipboard integration, native Unicode/IME decoding, focus
loss, and complete Go panels still need their platform and shell consumers.

## Context lifetime regression

The pinned imgui-rs revision `e246a334` calls `igEndFrame` on the current context
when dropping any context with a nonzero current frame count. The previous
suspended-context constructor allocated another context, failed activation, then
dropped that inactive allocation. This ended a different owner's live frame.

An isolated regression runs both a baseline frame and the conflicting constructor
case. Before the fix, baseline passed and refusal aborted the next widget with
`g.WithinFrameScope`. A debugger stopped at `ImGui::EndFrame`, called through
`Context::drop`, `SuspendedContext::drop`, and the constructor's `map_err`.

The owner now retains an active context for its whole lifetime. A private
read-only FFI check refuses occupied creation before allocation. The raw address
is not exposed or changed. The rest of the GUI crate denies unsafe code. All
context work stays on the single GUI thread; interleaved engines are not the
application ownership model. The suspended activation and retirement path was
removed, rather than retained as another strategy. Tests confirm live-frame
refusal, repeated frames, owner release, and packets surviving context destruction.
The dependency source and released kernel checkout were not modified.

## Kernel lowering

The GUI producer imports the viewport-owned packet types and has no direct
kernel dependency. Lowering stays in the existing viewport renderer boundary,
as the architecture gate requires. A retained `GuiFrameCompiler` converts the atlas once and shares the immutable
image across batches and frames. It converts sRGB vertex colors to linear floats,
keeps alpha unpremultiplied, and uses the kernel's textured draw pipeline.
Ordered GUI draws form a layer with depth test/write/clear disabled. Pixel
coordinates map to the kernel's OpenGL clip convention. A clip rectangle becomes
four intersected singleton plane chains; one four-plane union would be incorrect.

GPU tests check interior color against independent byte values, all four clipping
boundaries, and ordered alpha blending over a known background. They run with
Vulkan validation enabled. Dear ImGui tests select all four buttons through
press/release input, check framebuffer scaling, context refusal/release, and
owned snapshots after context destruction. This is the initial control and draw
boundary; the full Go ribbon, browser, dialogs, themes, assets, shortcut routing,
native input attachment, focus/IME/clipboard, and document-based enablement remain
open.

## Required native kernel composition contract

Released Martensite revision `193572dc` exposes `RenderFrame`/`RenderLayer`, but
`Graphic3dRenderView` builds native submissions through private `frame_for_view`.
Its public capture/redraw methods do not accept an owned overlay layer. An injected
`RenderBackend` is refused for native presentation in `driver_backend.rs`.
This is the remaining native seam for these packets under ADR-0008; the kernel
owner must release and test it.

The required extension attaches or clears an owned layer on an existing native
view. It must preserve the scene's layer ordering and clear color, put UI after
model rendering, preserve float colors/UVs and clip chains, and keep device,
swapchain, capture, and GPU retirement in the existing kernel owner. The packet's
expected extent must match the native target; a resize needs a fresh GUI frame.
It must not affect camera Fit bounds, model selection identity, or analytic
geometry. Font images must remain owned through GPU completion, including replace
and close. Kernel acceptance needs native pixels/capture, resize, replacement,
removal, and validation with the model and UI present together.

The application consumes tested owner releases. The current executable does not
yet attach these packets. The existing kernel text status overlay remains active.
This work does not close R1V/R4 or claim a usable CAD GUI.

## Local acceptance

```sh
VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers" \
  CARGO_NET_OFFLINE=true make gate
```

The workspace now contains thirteen crates, including the GUI frame producer.
Validation is local Linux only. No remote push, PR, or deferred-platform test is
included. The exact native composition requirement is recorded here for the
kernel owner; its active source and released checkout were not modified.

The latest full local gate passed with the software Vulkan driver selected by
`VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json`, using the same command
above. It covered all thirteen crates. Production line coverage was 98.74% for
GUI and 93.72% for viewport; duplication was 0.61%. All six GUI packet rendering
tests also passed on the AMD device with validation enabled.

The default RADV gate failed the native history pixel comparison. The same
failure was reproduced at the clean pre-change commit; see the
[native presentation diagnostic](diagnostic-native-present-pixels.md).
The hardware acceptance remains open. A disposable
`experiments/imgui-frame-preview` harness generated an actual Dear ImGui frame,
destroyed its UI context, and rendered the retained packet with the released
Vulkan renderer. The resulting offscreen control panel was visually inspected;
text orientation, font UVs, button colors, and clipping were correct, and Vulkan
validation reported no errors. This preview does not attach UI to the executable's
native view; the composition requirement above remains open.
