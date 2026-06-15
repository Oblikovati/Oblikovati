# ADR-0017 — Channel-based release pipeline (nightly + stable; GUI head + CLI)

**Status:** accepted (user decision, 2026-06) · **Supersedes:** the CD section of
[ADR-0015](ADR-0015-build-ci-and-test-tooling.md) (goreleaser on a `v*` tag).
Operational guide: [`RELEASING.md`](../../RELEASING.md).

## Context

ADR-0015 set up CD as **goreleaser on a `v*` tag**, cross-building the two
**cgo-free** binaries. We now also ship the **cgo + Vulkan GUI head**
(`oblikovati-head`) across Windows, Linux, and macOS, and want two delivery
channels driven by the branch model rather than hand-cut tags.

## Decision

1. **Two channels.**
   - **Nightly** — a rolling `nightly` **prerelease** built from `develop` (daily
     cron + manual dispatch; skipped when there are no new commits since the last
     nightly).
   - **Stable** — cut when a PR is merged **`develop` → `release`** (push to
     `release`); tagged `v{MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH}`.

2. **Versioning `{MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH}`** (revised 2026-06;
   superseded the earlier `MAJOR.MINOR.<timestamp>`). MANUAL_MAJOR is hand-set in the
   repo-root `version.yaml` (a deliberate generational bump, from 0); API_VERSION is the
   referenced `oblikovati.org/api` release, each component zero-padded to two digits and
   concatenated (`v0.2.0` → `000200`); MINOR/PATCH auto-number from conventional-commit
   scope (feat/breaking → minor, fix → patch) and reset to `0.0` when MANUAL_MAJOR or
   API_VERSION change. `cmd/obkversion` (tested `release` package) computes it from
   git tags + history; nightlies append `-nightly.<timestamp>`. The human only ever
   edits `version.yaml`.

3. **Artifacts.** The **GUI head + CLI** for **Windows**, **Linux (AppImage)**, and
   **macOS (Intel + Apple Silicon, unsigned** — no Apple cert yet). **Add-ins are
   not shipped** — they are maintained by external vendors (ADR-0003 / ADR-0016).

4. **Pipeline.** A reusable `build.yml` matrix: the cgo-free CLI is cross-compiled
   once; the cgo head is built **natively per OS** (it links system GLFW3 + Vulkan
   via `pkg-config`, so it cannot be cross-compiled) and packaged — Linux AppImage
   (`linuxdeploy`, Vulkan loader bundled, GPU ICD host-provided), Windows mingw zip,
   macOS tarball with MoltenVK + the loader bundled. `nightly.yml` / `release.yml`
   publish via `gh release`.

## Why (and why goreleaser is superseded)

- **goreleaser can't build the head.** It excels at cgo-free cross-compiles, but the
  cgo+Vulkan head needs per-OS native builds and platform packaging (AppImage,
  MoltenVK, mingw) it does not do. The pipeline is therefore a build matrix;
  goreleaser is kept only for local `--snapshot`.
- **Branch-driven channels** match the team's flow (develop = dev, release =
  stable) and remove hand-cut `v*` tags.
- **Timestamp PATCH** removes manual patch bookkeeping — the build number is the
  time — so `MAJOR.MINOR` is the only human-managed part and maps directly to
  API-compatibility intent.

## Consequences

- **`develop` is the default branch.** GitHub fires `schedule`/`workflow_dispatch`
  only from the default branch, so nightly requires it; this also matches the
  dev-on-develop model.
- **A push to `release` cuts a stable release** — treat merging `develop → release`
  as "ship it."
- Per-OS head packaging may need occasional CI tuning (AppImage Vulkan bundling,
  macOS dylib relocation, Windows DLLs); the CLI path is cgo-free and reliable, and
  the workflow ships it independently of head packaging.
- macOS packaging was later reworked by [ADR-0019](ADR-0019-macos-moltenvk-support.md):
  the head now ships as one **signed + notarized universal `.app`** (not an unsigned
  tarball), gated on `MACOS_*` secrets. Linux/Windows **arm64 head** is deferred (no
  standard arm runner) while the CLI ships all arches.
- CI (`ci.yml`) runs on `main` + `develop` and PRs; the release workflows are
  separate. ADR-0015's `make`/lint/test foundation is unchanged.
