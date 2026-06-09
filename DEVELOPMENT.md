# Oblikovati — developer guide

The Go implementation of the Oblikovati parametric CAD application. The
architecture lives in [`architecture/`](architecture); the backlog lives in
[`implementation-plan/`](implementation-plan). This is the developer workflow for
the code in this repository.

- **Module:** `oblikovati` (GPL-2.0). It `require`s the
  Apache-2.0 contract module `oblikovati.org/api` — now the sibling repo
  [`../Oblikovati.API`](../Oblikovati.API) — and **implements** it. The dependency
  is resolved for local development by the `go.work` workspace at the repo root
  (sibling checkouts), not a committed `replace`; see
  [ADR-0018](architecture/decisions/ADR-0018-apache-api-contract-module.md).
- **Go:** 1.22+ (see `go.mod`)
- **Core is cgo-free.** `math/`, `kernel/`, `model/` and friends build and test
  with `CGO_ENABLED=0`; cgo is confined to the renderer/platform edge
  ([ADR-0008](architecture/decisions/ADR-0008-cgo-boundary.md)).

## Local setup (go.work over sibling checkouts)

Check the contract repo out next to this one and tie them together with a
workspace (the `go.work` file is git-ignored — local dev only):

```sh
git clone https://github.com/Oblikovati/Oblikovati.git
git clone https://github.com/Oblikovati/Oblikovati.API.git   # sibling
cd Oblikovati
go work init . ./head
go work edit -replace oblikovati.org/api=../Oblikovati.API
```

(The sibling contract is wired in via `replace` rather than `use` so resolution
stays correct even if this module later gains an external dependency.)

## Public API: contract in ../Oblikovati.API, implementation here

The public API is its own module, **[`../Oblikovati.API`](../Oblikovati.API)**
(Apache-2.0). Anything that extends the public surface is two parts, in order
(ADR-0018):

1. **Define the contract in the API repo** — enums/value types in `api/types` (this
   module aliases them with `type X = types.X`), Go interfaces in `api/contract`,
   method-name constants + JSON DTOs in `api/wire`, a typed group in `api/client`.
2. **Implement here** — behavior in `kernel/`, `model/`, `app/`, or `head/`; a
   compile-time assertion that the impl satisfies the interface
   (`var _ contract.X = (*impl.X)(nil)`); and the handler wired into
   `addin/router` keyed on the `api/wire` method constant.

Never re-declare a wire DTO or method string here — import it from `api/wire`. The
API module never imports this module (CI enforces the boundary).

## Layout

The target package tree is defined in
[core/01-module-layout.md](architecture/core/01-module-layout.md). The Go
application module is at the repo root; the cgo Vulkan + ImGui GUI is the `head/`
submodule. `head/` vendors the C ABI header
(`head/internal/addinhost/include/oblikovati_addin.h`) so it builds standalone.

## Daily workflow

```sh
make tools      # one-time: install pinned golangci-lint + gotestsum
make test       # tier-1 cgo-free unit tests
make lint       # golangci-lint (house style from .golangci.yml)
make fmt        # format
make build      # version-stamped binaries into dist/
make ci         # everything CI runs: fmt-check vet lint test-race cover
make help       # all targets
```

Install the pre-commit hook (gofmt + vet + short tests) with `make hooks`.

## Build-time gating

Mode flags are compile-time constants in `build/`, toggled by build tags so dead
branches compile out:

```sh
go build -tags debug,profile,editor ./cmd/oblikovati
```

`build.Debug`, `build.Profile`, `build.Editor`. Version metadata
(`build.Version/Commit/Date`) is injected via `-ldflags` by the Makefile, which
derives the version from the repo-root `VERSION` file (`MAJOR.MINOR`) plus a
timestamp via `scripts/version.sh`; the release workflows pass the exact version.

## Conventions

- Unfinished paths return `build.NotYetImplemented("PBI-xxx")` — greppable and
  runtime-visible — never bare `TODO`/`FIXME`.
- Structured `log/slog` for diagnostics; no `fmt.Print*` (the linter forbids it).
- Errors are typed values; modeling failures are health state, not panics.
- Acceptance criteria are executable tests; metamorphic/property tests preferred
  (see [testing strategy](architecture/testing/README.md)).

## CI / CD

- **CI** ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)): lint, the
  cgo-free test matrix (Linux/macOS/Windows), the race detector, a cross-compile
  matrix, the SPDX check, and docs lint/links — on every push and PR. Go jobs check
  out `../Oblikovati.API` and point the `api` require at it.
- **Release** — two channels: a **nightly** prerelease from `develop`
  ([`nightly.yml`](.github/workflows/nightly.yml)) and a **stable** release on merge
  to `release` ([`release.yml`](.github/workflows/release.yml)), both building the
  GUI head + CLI via the reusable [`build.yml`](.github/workflows/build.yml) matrix.
  Versioning and the full process: [`RELEASING.md`](RELEASING.md).

Rationale: [ADR-0015](architecture/decisions/ADR-0015-build-ci-and-test-tooling.md)
(build/CI/test) and [ADR-0017](architecture/decisions/ADR-0017-release-pipeline.md)
(release pipeline).
