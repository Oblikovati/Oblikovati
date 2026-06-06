# ADR-0015 — Build, CI, and test tooling

**Status:** accepted; **CD section superseded by
[ADR-0017](ADR-0017-release-pipeline.md)** (channel-based release pipeline).

> **Amendment (2026-06, [ADR-0017](ADR-0017-release-pipeline.md)).** The CD
> decision below (goreleaser on a `v*` tag) was replaced once we began shipping the
> cgo+Vulkan GUI head: goreleaser can't build it per-OS. Releases are now a
> branch-driven build matrix — **nightly** prereleases from `develop` and **stable**
> releases on merge to `release`, versioned `MAJOR.MINOR.<timestamp>` (see ADR-0017
> and [`RELEASING.md`](../../RELEASING.md)). The `make`/lint/test/CI parts of this
> ADR are unchanged.

**Context:** greenfield Go implementation begins; we need
a delivery and test-automation foundation before feature code, so every later PBI
lands behind the same gates. Builds on [ADR-0001](ADR-0001-go-language.md) (Go),
[ADR-0008](ADR-0008-cgo-boundary.md) (cgo confined to the render/platform edge),
and the [testing strategy](../testing/README.md).

## Decision

Stand up the toolchain now, rooted at `/source` (per CLAUDE.md), module
`oblikovati`:

- **Task runner: `make`.** A single `Makefile` is the one command surface for
  build/test/lint/cover/release, so humans and CI invoke identical steps. No
  third-party runner dependency.
- **Build gating via `build/`.** Mode flags (`Debug`, `Profile`, `Editor`) are
  tag-selected compile-time constants (on/off file pairs); version metadata is
  link-time `-ldflags`. Realizes core/01's build-gating section.
- **`NotYetImplemented(issueID)`** lives in `build/` — the sanctioned, greppable,
  runtime-visible stub for the many kernel gaps during phasing.
- **Lint: `golangci-lint`** (pinned), config encodes the CLAUDE.md house style a
  linter can enforce: `funlen`/`gocyclo` (short, single-purpose functions),
  `revive` exported-docs, `errcheck`, `gofumpt`/`goimports`, and `forbidigo`
  banning `fmt.Print*` (structured `slog` only).
- **Tests: `gotestsum`** for JUnit output in CI; stdlib `testing` otherwise. The
  suite stays dependency-free for now so it builds and runs offline.
- **CD: `goreleaser`** cross-builds both binaries and publishes a GitHub release
  on a `v*` tag. *(Superseded by [ADR-0017](ADR-0017-release-pipeline.md): a
  channel-based build matrix that also ships the GUI head; goreleaser is kept only
  for local `--snapshot`.)*

## CI shape (implements testing/README "CI shape", Tier 1)

`/.github/workflows/ci.yml`, working-directory `source`, on every push/PR:

| Job | What | Why |
|---|---|---|
| `lint` | golangci-lint | house style as a hard gate |
| `test` | `CGO_ENABLED=0` matrix: Linux/macOS/Windows | the cgo-free 90% is portable & fast; proves it on all OSes |
| `race` | `go test -race` (Linux, cgo) | concurrency correctness (worker-pool recompute, ADR-0007) |
| `build` | cross-compile matrix (linux/darwin/windows × amd64/arm64) | the core must cross-build with no native toolchain (ADR-0008) |

Deferred until the relevant code exists (not stubbed as dead workflows):
**Tier 2** (CPU-reference renderer differentials, gRPC dogfood) and **Tier 3**
(software-Vulkan/Mesa-lavapipe renderer tests, Blender-oracle goldens, perf/soak)
from the testing strategy. They attach to this same workflow as the renderer
(M-renderer) and API land.

## Why these and not alternatives

- **make over Taskfile/mage** — zero install, ubiquitous, and the CD/test logic is
  thin; a Go-based runner buys nothing yet.
- **goreleaser over hand-rolled cross-compile + upload** — it already does the
  matrix, archives, checksums, and changelog the cgo-free build makes trivial.
- **No external test/assert deps yet** — keeps the tree buildable offline and the
  failure signal in our own code; property-test libraries get adopted with M01/M06
  when there is something to assert against.

## Consequences

- `make ci` locally == the CI gates; a green local run predicts a green pipeline.
- Coverage is measured every run with a `COVER_MIN` gate (currently 0, raised as
  the suite grows) so the threshold ratchets rather than regresses.
- Adding a package needs no tooling change; it inherits lint, the test matrix,
  cross-build, and release automatically.
- The renderer's heavier oracle tiers are documented-but-deferred, avoiding dead
  CI jobs while the design (testing/00) is already settled.
