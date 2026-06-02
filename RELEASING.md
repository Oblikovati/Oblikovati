# Releasing Oblikovati

Two channels, both automated by GitHub Actions:

| Channel | Trigger | Workflow | Result |
|---------|---------|----------|--------|
| **Nightly** | daily cron + manual `workflow_dispatch`, on `develop` | `.github/workflows/nightly.yml` | rolling `nightly` **prerelease** (skipped if no new commits) |
| **Stable** | push to `release` (merge a PR `develop` → `release`) | `.github/workflows/release.yml` | GitHub release tagged `vMAJOR.MINOR.<timestamp>` |

Both build the **GUI head + CLI** for Windows, Linux (AppImage), and macOS (Intel +
Apple Silicon, **unsigned** — right-click → Open the first time). Add-ins are **not**
shipped; they are maintained by external vendors.

## Versioning — `MAJOR.MINOR.PATCH`

- **MAJOR** — a **breaking change to the public API** (starts at `0`).
- **MINOR** — a **non-breaking API extension _or_ a bug fix**.
- **PATCH** — the **build number**: a UTC timestamp (`YYYYmmddHHMMSS`), set
  automatically. Nightlies add a `-nightly` suffix (a semver prerelease).

`MAJOR.MINOR` lives in the repo-root [`VERSION`](VERSION) file. `scripts/version.sh`
appends the timestamp:

```sh
scripts/version.sh stable   # -> 0.0.20260602T120000   (e.g.)
scripts/version.sh nightly  # -> 0.0.20260602T120000-nightly
```

### Bumping `MAJOR.MINOR`

Edit `VERSION` **in the PR that introduces the change**, per the rules above:

- breaking API change → bump MAJOR (e.g. `0.7` → `1.0`), reset MINOR to `0`;
- API extension or bug fix → bump MINOR (e.g. `0.7` → `0.8`).

PATCH is never edited by hand — it is the build timestamp.

## Cutting a stable release

1. Land your changes on `develop` (with the `VERSION` bump if warranted).
2. Open and merge a PR **`develop` → `release`**.
3. The `release` workflow builds all platforms and publishes
   `vMAJOR.MINOR.<timestamp>` with the binaries + `checksums.txt` + generated notes.

The `release` branch is the stable channel (one-time setup: `git branch release &&
git push -u origin release`).

## Local builds

`make build` / `make build-all` (in `source/`) stamp `build.Version` from `VERSION`
+ a timestamp, matching the release scheme. CI overrides it with the exact release
version via `VERSION=… make build`.

## Notes / known follow-ups

- Per-OS head packaging (AppImage Vulkan-loader bundling, macOS MoltenVK relocation,
  Windows mingw DLLs) is built in `scripts/package-*.sh` and the `build` workflow; it
  needs a few CI iterations to harden across platforms. The CLI path is cgo-free and
  reliable.
- macOS builds are **unsigned** (no Apple Developer cert yet) — Gatekeeper will warn.
- Linux/Windows **arm64 head** is deferred (no standard arm CI runner); the CLI ships
  for arm64 on all OSes.
- `source/.goreleaser.yaml` is superseded by this pipeline (kept only for local
  `goreleaser --snapshot`).
