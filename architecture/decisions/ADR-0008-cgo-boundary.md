# ADR-0008 — Confine cgo to the platform/render edge

**Status:** accepted · **Context:** ADR-0002 mandates a pure-Go kernel, but a
Vulkan app needs a window, a surface, and the Vulkan loader.

## Decision

`kernel/`, `model/`, `math/`, `parameters/`, `persistence/` are **pure Go, cgo-free**
— they cross-compile and unit-test with no native toolchain. Any cgo is **confined
to `platform/` and `renderer/`** (windowing, input, the Vulkan loader/surface),
behind interfaces, in `*_cgo.go` / platform-suffixed files.

Two acceptable implementations of that edge, chosen by build tag:

1. **Pragmatic (default): cgo.** GLFW (`go-gl/glfw`) for window/input/surface +
   a cgo Vulkan binding. Mature, well-trodden, best driver compatibility.
2. **Purist (optional): purego.** `ebitengine/purego` to `dlopen` libvulkan and
   the platform windowing libs **with no cgo at all**, making the *entire* binary
   cgo-free and cross-compilable. More upfront binding work; keeps ADR-0002's
   "no cgo" spirit end-to-end.

We start with (1) for velocity and keep the edge narrow enough that (2) is a
drop-in later if cgo cross-compilation friction justifies it.

## Why confine it

- **Cross-compilation & CI.** The valuable 90% of the codebase — the entire domain
  model and kernel — stays `CGO_ENABLED=0`, so it builds for all targets trivially
  and runs in headless CI without GPUs or OS UI libs.
- **Testability.** The model/kernel are tested without linking native code; the
  renderer is tested separately (and can be swapped for a headless/offscreen or
  null backend, reusing the render-target abstraction — realtime-3d §4).
- **Portability seam.** If macOS Vulkan (MoltenVK, ADR-0005) or a future Metal/
  WebGPU backend is wanted, only `renderer/` changes.

## The interface

```go
// platform/  — implemented by *_cgo.go (GLFW) or *_purego.go
type Window interface {
    Surface(instance vk.Instance) (vk.SurfaceKHR, error)
    Size() (w, h int)
    PollEvents()
    Events() <-chan input.Event
    Close()
}
```

App code (and the renderer's higher layers) depend only on `platform.Window` and
the `renderer` interfaces — never on GLFW or a raw `Vk*` type (realtime-3d §4).

## Consequences

- `go test ./kernel/... ./model/... ./math/...` runs anywhere, fast, no cgo.
- A `null`/offscreen renderer backend exists for tests and thumbnail/headless
  export (reused by drawing thumbnails, M14, and CI screenshot diffs).
- Build matrix documents `CGO_ENABLED` per package set in `build/`.
