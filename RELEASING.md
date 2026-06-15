# Releasing Oblikovati

Two channels, both automated by GitHub Actions:

| Channel | Trigger | Workflow | Result |
|---------|---------|----------|--------|
| **Nightly** | daily cron + manual `workflow_dispatch`, on `develop` | `.github/workflows/nightly.yml` | rolling `nightly` **prerelease** (skipped if no new commits) |
| **Stable** | push to `release` (merge a PR `develop` → `release`) | `.github/workflows/release.yml` | GitHub release tagged `v{MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH}` |

Both build the **GUI head + CLI** for Windows, Linux (AppImage), and macOS (one
**universal `.app`**, codesigned + notarized — see [macOS signing](#macos-signing--notarization)).
Add-ins are **not** shipped; they are maintained by external vendors.

## Versioning — `{MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH}`

- **MANUAL_MAJOR** — a deliberate generational bump, hand-set in the repo-root
  [`version.yaml`](version.yaml) (`major:`); starts at `0`.
- **API_VERSION** — the referenced `oblikovati.org/api` release (the `require` in
  `go.mod`, kept current automatically), each semver component zero-padded to two
  digits and concatenated: `v0.2.0` → `000200`.
- **MINOR** / **PATCH** — auto-numbered from the **conventional-commit scope** since the
  last stable release, per SemVer: a `feat` (or a breaking change) bumps **MINOR** and
  resets PATCH; a `fix` bumps **PATCH**. They **reset to `0.0`** whenever MANUAL_MAJOR or
  API_VERSION changes (a new line gets its own sequence). Nightlies append
  `-nightly.<timestamp>` (a semver prerelease).

[`cmd/obkversion`](cmd/obkversion) computes the whole string; its logic lives in the
tested [`release`](release) package. It reads `version.yaml`, the api pin, and git
tags + commit history — so it needs a full checkout (`fetch-depth: 0`):

```sh
go run ./cmd/obkversion stable    # -> 0.000200.1.0          (e.g.)
go run ./cmd/obkversion nightly   # -> 0.000200.1.0-nightly.20260602T120000
```

### Bumping the version

- **MINOR / PATCH** are never edited by hand — use [Conventional
  Commits](https://www.conventionalcommits.org) (`feat:` / `fix:` / `feat!:`) and the
  pipeline derives the bump.
- **API_VERSION** changes on its own when the `oblikovati.org/api` pin is bumped (CI
  tracks the latest contract release).
- **MANUAL_MAJOR** — edit `version.yaml` only for a deliberate generational release.

## Cutting a stable release

1. Land your changes on `develop` (conventional-commit messages drive the bump).
2. Open and merge a PR **`develop` → `release`**.
3. The `release` workflow computes the version, builds all platforms, and publishes
   `v{MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH}` with the binaries + `checksums.txt`
   + generated notes. That tag becomes the base for the next release's MINOR.PATCH.

The `release` branch is the stable channel (one-time setup: `git branch release &&
git push -u origin release`).

## macOS signing & notarization

The macOS head ships as a single **universal `Oblikovati.app`** (Intel + Apple Silicon
lipo'd together) that is Developer-ID **codesigned, notarized, and stapled**, so a
downloaded build runs on a clean Mac with **no Vulkan SDK, no environment setup, and no
Gatekeeper prompt**. MoltenVK + the Vulkan loader + GLFW are bundled in
`Contents/Frameworks` (found via `@rpath`); the app points the loader at the bundled
MoltenVK ICD in-process at startup (`head/internal/native/icd_darwin.go`).

This requires an Apple Developer account. Add these **repository secrets** (Settings →
Secrets and variables → Actions). When any is missing, the macOS release is **skipped**
(the rest of the build still publishes):

| Secret | What it is |
|--------|------------|
| `MACOS_CERT_P12_BASE64` | base64 of the **Developer ID Application** certificate exported as `.p12` (`base64 -i cert.p12 \| pbcopy`) |
| `MACOS_CERT_PASSWORD` | password set when exporting the `.p12` |
| `MACOS_AC_APPLE_ID` | Apple ID email used for notarization |
| `MACOS_AC_PASSWORD` | an **app-specific password** for that Apple ID (appleid.apple.com → Sign-In and Security) |
| `MACOS_AC_TEAM_ID` | the 10-character Apple Developer **Team ID** |

The signing identity is derived automatically from the imported certificate. The
build/sign/notarize flow lives in `scripts/package-macos.sh` + `scripts/macos-sign.sh`
and the `head-macos*` jobs in `build.yml`. Notarization adds a few minutes (Apple-side
scan); `notarytool --wait` blocks the job on the verdict.

## Local builds

`make build` / `make build-all` stamp `build.Version` via `go run ./cmd/obkversion
stable`, matching the release scheme. CI overrides it with the exact release version
via `VERSION=… make build`.

## Notes / known follow-ups

- Per-OS head packaging (AppImage Vulkan-loader bundling, macOS MoltenVK relocation,
  Windows mingw DLLs) is built in `scripts/package-*.sh` and the `build` workflow; it
  needs a few CI iterations to harden across platforms. The CLI path is cgo-free and
  reliable.
- macOS builds are codesigned + notarized when the `MACOS_*` secrets are set (see
  [macOS signing](#macos-signing--notarization)); otherwise the macOS release is skipped.
- Linux/Windows **arm64 head** is deferred (no standard arm CI runner); the CLI ships
  for arm64 on all OSes.
- `source/.goreleaser.yaml` is superseded by this pipeline (kept only for local
  `goreleaser --snapshot`).
