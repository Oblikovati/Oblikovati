# Martensite viewport integration — partial R1V

Tracking: [#3630](https://github.com/Oblikovati/Oblikovati/issues/3630).
The application boundary remains [ADR-0008](../decisions/ADR-0008-vulkan-renderer.md).
This is new implementation evidence, not a replacement for the accepted ADR.

## Checked source and Linux runtime

On 2026-10-10, the local RustyCADKernel checkout was clean at
`ead13bcb5d2313236e61b15a22a127a23cd8bdd7`, including native presentation merge
`90fce3665`. The application port independently ran this command in that checkout
with the existing Linux X11 display `:1`:

```sh
cargo run -p martensite-vulkan --example window_acceptance --locked
```

The command exited successfully and reported:

```text
MARTENSITE_DEVICE=AMD Radeon 8060S Graphics (RADV STRIX_HALO)
```

The inspected example requires Vulkan validation and checks its error list is
empty. It compares actual X11 drawable pixels with completed GPU captures across
six window modes, 24 alternating resizes, vertical-sync changes, window
replacement, unmap/remap, simultaneous views, view removal, and image dumps that
must not change displayed pixels. This run verifies the kernel example on the
native Linux GPU. It does not verify an Oblikovati window, an application scene,
Dear ImGui composition, picking, or complete legacy rendering parity. No Windows
or macOS checks were run, and their application pipelines remain unchanged.

## Integration boundary

The viewport can use `Graphic3dRenderDriver`, `V3dViewer`, and `V3dView` to submit
retained presentations. `XwWindow` and `AspectDisplayConnection` provide the Linux
window connection. The application must define its window lifetime and event
loop, then bind the window with `set_window`, notify size changes with
`must_be_resized`, and remove views before releasing their owners.

Use `Graphic3dRenderView::try_redraw` and `capture` to observe failures rather
than treating the void redraw callback as success. Associate returned captures
with application view and scene generations. Martensite owns the device, queues,
swapchains, fences, and presentation-resource retirement; the application must
not create another swapchain for the same view.

The current public renderer exports do not provide a Dear ImGui composition
interface. That part of R1V remains open. It needs a kernel-owned composition
contract and tests for shared submission, texture lifetime, clipping, and resize.
The application must not fill this gap with a second Vulkan renderer.

The native window path additionally requires the surface/Xlib surface,
surface-capabilities2, surface-maintenance1, swapchain, and swapchain-maintenance1
extensions/features described by the kernel. Vulkan 1.3 alone is insufficient.
Unsupported surfaces must produce a visible error; no silent offscreen success
can stand in for native presentation.

## Package availability

`cargo package -p martensite-vulkan --list --locked` succeeds at the checked
revision. Its inventory includes all eight embedded SPIR-V modules, their shader
sources, the window acceptance example, and its support sources. This is a file
inventory check, not an extracted-package build or a release certification.
The owner release tooling derives the complete crate set from Cargo metadata and
includes `martensite-vulkan`; keep its sibling crates, lockfile, license, and
native runtime requirements together under the existing release contract.

On a subsequent fetch on 2026-10-10, remote kernel `main` reached
`ead13bcb5d2313236e61b15a22a127a23cd8bdd7`, matching the checked rendering
revision. The repository still had no published GitHub releases. Source
publication is confirmed; release-package verification remains open. Do not replace the
application's released-dependency checks with a sibling path or a machine-local
Git URL. Once the kernel owner publishes the packages, pin that
immutable source in the viewport crate and verify the extracted package set.
The application release pins remain unset; publication is still blocked.

## Remaining acceptance

- Add the application viewport adapter against the published kernel source.
- Prove application-owned window creation, events, resize, capture, removal, and
  shutdown on Linux, with explicit failure handling.
- Add the shared Dear ImGui composition interface in Martensite and consume it.
- Exercise application scenes and captures through the foreground host before
  claiming R1V complete. Later R4/R7 checks cover modeled shapes, picks, styles,
  appearance, and lighting.
