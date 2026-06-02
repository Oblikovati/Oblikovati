# ADR-0019 — macOS support: MoltenVK runtime fixes + signed universal .app

**Status:** accepted (2026-06) · **Updates:** [ADR-0017](ADR-0017-release-pipeline.md)
(macOS artifact: unsigned tarball → signed universal `.app`). **Builds on:**
[ADR-0005](ADR-0005-vulkan13-renderer.md) (Vulkan 1.3), [ADR-0008](ADR-0008-cgo-boundary.md)
(cgo edge). Operational guide: [`RELEASING.md`](../../RELEASING.md).

## Context

The head (Vulkan 1.3 + GLFW + Dear ImGui) was developed and CI-tested on Linux/X11,
where Vulkan is a system facility. The first run on a Mac host showed the head did not
start **at all** on macOS, and the existing macOS release (ADR-0017: an unsigned tarball
with a `DYLD_*`-setting launcher) could never give a clean-Mac, no-workaround download.
macOS differs from Linux/Windows in four ways that each broke a different stage:

1. **No system Vulkan.** Vulkan reaches the GPU only through **MoltenVK** (Vulkan→Metal),
   a *portability* driver that ships with the app, not the OS.
2. **The Vulkan loader hides portability drivers** unless the instance explicitly opts in.
3. **Cocoa requires the main thread** for all window/event calls.
4. **Gatekeeper blocks downloaded code** that is not Developer-ID signed *and* notarized,
   and the **hardened runtime** notarization requires **strips `DYLD_*`** on exec.

## Decision

Make the head run on MoltenVK and ship it as a self-contained, notarized app. All
runtime changes are guarded so Linux/Windows behavior is unchanged.

### Runtime (head)

1. **Opt into portability enumeration** in `vkCreateInstance`: add
   `VK_KHR_portability_enumeration` + the `ENUMERATE_PORTABILITY_BIT_KHR` flag. Without
   it the loader reports *"Found no drivers!"* and `vkCreateInstance` fails with
   `VK_ERROR_INCOMPATIBLE_DRIVER` — MoltenVK is a portability device the loader will not
   enumerate unless asked.
2. **Enable `VK_KHR_portability_subset`** on the device when the GPU advertises it —
   mandatory for a portability device, else device creation is a validation error
   (`VUID-VkDeviceCreateInfo-pProperties-04451`).
3. **Pin the main OS thread** (`runtime.LockOSThread` in `native`'s package `init`), so
   GLFW init/window/poll run on the main thread Cocoa demands. Inert on other platforms.
4. **Point the loader at the bundled MoltenVK ICD in-process** at startup
   (`head/internal/native/icd_darwin.go` sets `VK_ICD_FILENAMES` before `vkCreateInstance`).
   In-process `setenv` is honored by the loader (cgo syncs Go env → libc) and, unlike a
   launcher's `DYLD_*`, **survives the hardened runtime**. Guarded: it does nothing
   unless the bundled manifest exists and no override is already set, so `go run`/tests/
   Linux are unaffected.

Extensions 1–2 are added only when the loader/device advertises them (queried at runtime),
so the same source is correct on every platform.

### Distribution (CI)

Replace the unsigned tarball + `DYLD_*` launcher with a **single universal
`Oblikovati.app`**, Developer-ID **codesigned + notarized + stapled**:

- **Bundle** the loader + MoltenVK + GLFW in `Contents/Frameworks`; the binary finds
  them via a baked-in **`@rpath`** (`@executable_path/../Frameworks`), not `DYLD_*`.
- **GLFW finds the loader itself** via its Cocoa bundle search (the app is a real
  `.app` with `libvulkan.1.dylib` in `Frameworks`), so no `DYLD_LIBRARY_PATH`.
- **One Team ID for everything.** The hardened runtime's *library validation* requires
  every loaded dylib to share the main binary's Team ID, so CI signs the binary and all
  bundled dylibs with the same Developer ID — which removes the need for the insecure
  `disable-library-validation` entitlement.
- **Universal**: each arch (Intel + Apple Silicon) is built natively (the cgo head can't
  cross-compile, per ADR-0008/0017) and `lipo`'d into one `.app`.
- **Gated on secrets**: the whole macOS release is **skipped** (not failed) when the
  `MACOS_*` signing secrets are absent, so forks/unconfigured repos still build the rest.

Files: `head/internal/native/{icd_darwin.go,mainthread.go,app.cpp}`,
`scripts/{package-macos.sh,macos-stage.sh,macos-sign.sh}`, and the `macos-preflight` /
`head-macos-build` / `head-macos` jobs in `build.yml` (`release.yml` + `nightly.yml`
pass `secrets: inherit`).

## Why (alternatives rejected)

- **Why a `.app` + in-process `setenv`, not the `DYLD_*` launcher (ADR-0017)?** The
  launcher is fundamentally incompatible with notarization: the hardened runtime strips
  `DYLD_*`, so a notarized launcher build could not find its libraries. The `.app` +
  `@rpath` + in-process ICD path is the only design that is both self-contained and
  notarizable. Verified end-to-end on a clean environment before wiring CI.
- **Why bundle the loader, not link MoltenVK directly?** Keeping the loader keeps one
  code path with Linux/Windows and preserves the option of validation layers in dev;
  the portability requirement is a *loader* behavior we now satisfy correctly anyway.
- **Why notarize, not just sign?** Developer-ID signing alone still trips Gatekeeper on
  downloaded apps (macOS 10.15+); notarization + stapling is what removes the prompt.

## Consequences

- **ADR-0017's macOS artifact is updated**: unsigned per-arch tarball → one signed,
  notarized universal `.app`; `RELEASING.md` lists the required `MACOS_*` repo secrets.
- **macOS dev still needs setup** (brew GLFW/loader/MoltenVK + a `go.work`); the bundle
  machinery is release-only — see `DEVELOPMENT.md` / the dev notes.
- One **benign validation message remains** (`UNASSIGNED-non-acquired-swapchain-image-used`
  at startup) in the shared swapchain loop, not MoltenVK-specific; tracked as a
  low-priority follow-up.
- First notarization may surface an entitlement need (Metal/MoltenVK); entitlements are
  intentionally minimal and a one-line addition in `macos-sign.sh` if required.
