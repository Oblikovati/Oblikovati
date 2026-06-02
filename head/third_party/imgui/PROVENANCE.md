# Vendored Dear ImGui

Dear ImGui **v1.92.8 WIP** core + the GLFW and Vulkan backends, vendored here and
compiled by our cgo `head` package against our own `imconfig.h`.

- License: MIT (Dear ImGui, © Omar Cornut). Upstream: https://github.com/ocornut/imgui
- Source of these exact files: the `cwrappers/imgui` tree bundled in
  `github.com/AllenDang/cimgui-go@v1.5.0` (pristine upstream sources).

## Why vendored, not cimgui-go

cimgui-go ships a *prebuilt* `cimgui.a` built with a generated `imconfig.h` that
injects interop types (`MyVec2`/`MyVec4`/`MyMatrix44`) which are not defined in the
distributed module — so compiling the Vulkan backend ABI-compatibly against that
archive is not reproducible. Vendoring the pristine sources and supplying our own
clean `imconfig.h` makes the ImGui ABI fully self-consistent and under our control.

## Files

- `imgui.{h,cpp}`, `imgui_draw.cpp`, `imgui_tables.cpp`, `imgui_widgets.cpp`,
  `imgui_internal.h`, `imstb_*.h` — ImGui core.
- `backends/imgui_impl_glfw.{h,cpp}` — GLFW platform backend.
- `backends/imgui_impl_vulkan.{h,cpp}` — Vulkan renderer backend.
- `imconfig.h` — **ours**, not upstream's.

## Updating

Bump by copying the same file set from a newer pristine Dear ImGui, keeping our
`imconfig.h`. Re-run the head smoke test after any bump.
