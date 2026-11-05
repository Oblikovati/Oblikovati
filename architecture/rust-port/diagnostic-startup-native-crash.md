# Startup integration: local validation-layer failure

Date: 2026-10-12. Worktree base: local `31ba4965` on `rust-port`.
No push, PR, remote test, or kernel source change was made. Full application
acceptance remains open. The local gate uses an official replacement validation
layer; the private diagnostic patch is not an application dependency.

## Implemented work

The Linux head accepts native document paths, frame options, and `--` for
literal operands. Startup uses the API typed documents client with visible
opens before the first owner frame. Errors name the failing path, stop later
opens, and retire the owner before the window. Core parser, named-fake caller,
CLI executable, native scene, and existing head/viewport suites have passed.
A real stored prism is displayed and its source bytes remain unchanged.

Native Xlib loading now retains one process-wide initialized library. Its
thread initialization completes before application display connections open.
The concurrent-library test checks shared ownership. This enforces the
[Xlib thread contract](https://xorg.freedesktop.org/archive/current/doc/libX11/libX11.html),
but does not fix the separate Vulkan crash described below.

## Captured failure

Repeated combined head unit tests intermittently segfault. A temporary signal
trace identifies this call chain, before startup documents are opened:

```text
NativeHeadWindow::open
ViewportRuntime::new
Graphic3dRenderDriver::new
VulkanRenderer::new
RasterPipelines::new
DrawPipeline::new / create
create_pipeline
ash::Device::create_graphics_pipelines
libVkLayer_khronos_validation.so
```

The installed layer advertises Vulkan API 1.3.275. Library build ID:
`62c351be72273187a6bfad702dd668de3a89e4e7`.
The failing layer offsets are `0x160aef5`, `0x160e2be`, and `0x1618b71`.
Martensite remains v0.2.1, `193572dc3665598c4948777c87375bad7dc8114a`.
Other repetitions failed on the existing three-second native window mapping
bound. These are distinct observations; neither is accepted or suppressed.

## Reproduction and narrowing evidence

Diagnostic artifacts are outside the application under
`../experiments/x11-navigation-crash` and `../experiments/viewport-init-stress`.
The temporary trace library is `/tmp/oblikovati-x11-trace.so`.
The combined traced head loop reproduced the segmentation fault; its logs are
`/tmp/oblikovati-startup-full-trace-*.log`. Untraced combined repeats also fail.
`addr2line` resolves the application/kernel frames listed above.

Native-only head tests passed 24 repetitions. A standalone harness passed
128 concurrent renderer constructions, 128 concurrent native window/render
lifecycles, and 256 rolling native lifecycles. These results do not reproduce
the combined-test fault and therefore do not prove a fix. The combined test
set, scheduling, or runtime state remains relevant to further minimization.

## Debugger evidence and comparison

An isolated GDB 15.1 and the exact Ubuntu debug symbols caught a second fault
in `GetLayerDataPtr<ValidationObject>`, called by the layer's `CreateImage`.
The debugger found an invalid bucket node in the shared `layer_data_map`.
Two core threads were reading that map when they faulted. The official
[v1.3.275 container source](https://github.com/KhronosGroup/Vulkan-ValidationLayers/blob/v1.3.275/layers/containers/custom_containers.h)
uses unprotected map lookup/insertion and erase, including a TODO to add a lock.
The chassis uses this map before per-object validation locks.

The original hypotheses were layer concurrency, invalid kernel pipeline inputs,
and creation overlapping teardown. Disabling shader caching in a temporary
layer settings file produced 20 green runs, but did not establish a fix.
Coarse validation locking still crashed. Neither setting was adopted.

A private source build of v1.3.275 added one shared mutex around map lookup,
insertion, and erase. It passed 60 combined-suite runs: 2,760 head unit test
executions, with validation enabled and native tests concurrent. This supports
the map race diagnosis. The build and patch remain clearly marked diagnostic
artifacts under `../experiments/vvl-1.3.275`; they are not installed or shipped.

The official Ubuntu `vulkan-validationlayers_1.4.321.0-1_amd64.deb` passed
100 combined-suite runs: 4,600 head unit test executions. Loader diagnostics
confirmed its absolute library path, so these were not accidental runs with
the old system library. That version replaces the old global-map chassis.
A finite stress run does not prove the absence of all native races.

## Local test toolchain

The [official Ubuntu package](https://archive.ubuntu.com/ubuntu/pool/universe/v/vulkan-validationlayers/vulkan-validationlayers_1.4.321.0-1_amd64.deb)
has SHA-256 `8e0b69e326e4ef7b9cc22008e757a75111289e3c956d01a493642f53d8111533`.
This matched the Questing universe package index. The index hash matched
`InRelease`, whose signature verified with the installed Ubuntu archive keyring
and signing key `F6ECB3762474EDA9D21B7022871920D1991BC93C`.

The extracted official library and manifest are kept under ignored
`.tools/vulkan-validation-1.4.321/`. Only the manifest's library path changes to
select the extracted library. [RELEASING](../../RELEASING.md) records the gate
command. System libraries, application validation defaults, test concurrency,
and the released kernel checkout are unchanged. No validation check is disabled.

The regression seam for this external failure is the concurrent combined head
suite, repeated against the selected layer. Application tests cannot directly
assert the internals of a third-party C++ map. Native-only tests and the smaller
harness did not reproduce it; they are not substitutes for the combined loop.
No application mutex, skipped test, or timing retry was introduced. The core and
debugger outputs remain local under `/tmp`; they must not be uploaded wholesale.
Do not describe the fault as pre-existing without a clean wave-base reproduction.

The full local Linux gate passed with the official layer for all 12 crates.
Every crate exceeded 80% production line coverage; duplication was 0.58%.
Shared ImGui composition and the complete application port remain pending.
