# Oblikovati

[![CI](https://github.com/Oblikovati/Oblikovati/actions/workflows/ci.yml/badge.svg?branch=develop)](https://github.com/Oblikovati/Oblikovati/actions/workflows/ci.yml)
[![nightly](https://github.com/Oblikovati/Oblikovati/actions/workflows/nightly.yml/badge.svg)](https://github.com/Oblikovati/Oblikovati/actions/workflows/nightly.yml)
[![release](https://github.com/Oblikovati/Oblikovati/actions/workflows/release.yml/badge.svg)](https://github.com/Oblikovati/Oblikovati/actions/workflows/release.yml)
[![latest release](https://img.shields.io/github/v/release/Oblikovati/Oblikovati?sort=semver)](https://github.com/Oblikovati/Oblikovati/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![App License](https://img.shields.io/badge/app-GPL_2.0-blue.svg)](LICENSE)
[![API License](https://img.shields.io/badge/api-Apache_2.0-blue.svg)](https://github.com/Oblikovati/Oblikovati.API)

**Oblikovati is a parametric, feature-based, history-driven mechanical-CAD (MCAD)
application** — an Inventor-class 3D solid modeler — rebuilt from the ground up in
**Go** with a **Vulkan 1.3** renderer, for **Linux, macOS, and Windows**.

It models a solid as an *editable program*, not a static mesh: **sketches → features
→ a recomputed B-rep**, driven by **dimensioned parameters** and a dependency graph,
with **persistent topological naming** (edits survive recompute), transactions/undo,
health/suppression, and an extensible **add-in** surface. The project is a
modernization of a mature Inventor-class COM automation API — *keep the domain model,
replace the Windows/COM plumbing with idiomatic Go and a modern GPU stack*
([architecture/README.md](architecture/README.md)).

Two binaries ship from this repo:

- **`oblikovati-head`** — the GUI: a Vulkan 1.3 viewport + Dear ImGui shell.
- **`oblikovati-cli`** — a headless, cgo-free command-line tool.

Plus the first add-in, in its own repo:

- **`oblikovati-mcp-bridge`** — an [MCP](https://modelcontextprotocol.io) server that
  lets an LLM drive the live application in real time, in
  [`../Oblikovati.AddIns`](https://github.com/Oblikovati/Oblikovati.AddIns).

> **Status — early/foundational.** Part modeling works today (sketches, parameters,
> the constraint solver, a B-rep kernel, and features such as extrude), along with
> the GUI head, the CLI, and the MCP add-in. Assemblies, drawings, sheet-metal, and
> the rest are on the [roadmap](implementation-plan/). Expect rapid change.

## Install

Download from the [**Releases**](https://github.com/Oblikovati/Oblikovati/releases)
page. Stable builds are versioned `vMAJOR.MINOR.<timestamp>`; **nightly** builds are
prereleases under the rolling `nightly` tag.

**GUI (`oblikovati-head`)** — needs a Vulkan-capable GPU/driver:

- **Linux** — download the `*.AppImage`, then `chmod +x` it and run. The Vulkan
  loader is bundled; your GPU's Vulkan driver is used from the system.
- **Windows** — download and unzip `*-windows-amd64.zip`, run `oblikovati-head.exe`
  (the GLFW + runtime DLLs are bundled; `vulkan-1.dll` comes from your GPU driver).
- **macOS** — download `*-darwin-<arch>.tar.gz`, extract, run `./oblikovati/oblikovati-head`.
  Vulkan runs over **MoltenVK** (bundled). Builds are **unsigned** (no Apple cert
  yet), so the first launch needs **right-click → Open** to clear Gatekeeper.

**CLI (`oblikovati-cli`)** — a single static binary for your OS/arch; download, make
it executable, and run. `oblikovati-cli version` prints the build metadata.

Verify downloads against `checksums.txt` (`sha256sum -c`).

## Develop

**Prerequisites**

- **Core + CLI:** Go 1.22+ (cgo-free).
- **GUI head:** Go 1.24, a C/C++ toolchain, and **GLFW 3 + Vulkan** dev libraries
  (the head links them via `pkg-config`). The Vulkan/GLFW packaging differs per OS —
  see the [release workflow](.github/workflows/build.yml) for the exact deps.

The Apache-2.0 contract is the sibling repo `Oblikovati.API`; check it out next to
this one and tie them together with a `go.work` workspace (git-ignored, local only).
See [DEVELOPMENT.md](DEVELOPMENT.md) for the full workflow.

```sh
git clone https://github.com/Oblikovati/Oblikovati.git
git clone https://github.com/Oblikovati/Oblikovati.API.git   # sibling
cd Oblikovati
go work init . ./head                                          # local workspace (go.work)
go work edit -replace github.com/Oblikovati/api=../Oblikovati.API

# Core + CLI (pure Go).
make test        # fast, cgo-free unit tests
make lint        # golangci-lint (house style)
make build       # version-stamped binaries into dist/
make ci          # everything CI runs locally: fmt-check vet lint test-race cover

# GUI head (cgo + Vulkan + GLFW) — separate module.
cd head
make run         # build + launch the windowed app
make smoke       # open a window, render a few frames, exit (CI smoke)
```

The MCP bridge add-in lives in [`Oblikovati.AddIns`](https://github.com/Oblikovati/Oblikovati.AddIns);
build it there and it loads into the head at runtime.

Docs are linted too: `make docs-lint` (at the repo root) runs markdownlint; CI also
link-checks the docs. House style and conventions live in [CLAUDE.md](CLAUDE.md).

## Documentation

| Area | Where | What it covers |
|------|-------|----------------|
| **Architecture** | [`architecture/`](architecture/README.md) | How the system is built: `core/` (runtime, object model, kernel, params, persistence, events, renderer, UI), `modeling/`, `assembly/`, `apps/`, `testing/`, and the **ADRs** in `decisions/`. |
| **Roadmap** | [`implementation-plan/`](implementation-plan/) | The 19-milestone / 170-PBI feature plan and live `PROGRESS.md`. |
| **Dev workflow** | [`DEVELOPMENT.md`](DEVELOPMENT.md) | Module layout, `go.work` setup, `make` targets, build-time gating, code conventions. |
| **Public API** | [`Oblikovati.API`](https://github.com/Oblikovati/Oblikovati.API) | The Apache-2.0 contract module (`types`, `contract`, `wire`, `client`) both the app and add-ins build on — [ADR-0018](architecture/decisions/ADR-0018-apache-api-contract-module.md). |
| **Add-ins** | [`Oblikovati.AddIns`](https://github.com/Oblikovati/Oblikovati.AddIns) | The proprietary add-ins (incl. the MCP bridge): the MCP tools/resources, the host JSON method contract, and the add-in C ABI. |
| **Releasing** | [`RELEASING.md`](RELEASING.md) | The nightly + stable channels and the `MAJOR.MINOR.PATCH` versioning rules. |
| **Reference API** | [`Oblikovati.Contracts`](https://github.com/Oblikovati/Oblikovati.Contracts) (archived) | The Inventor-class COM/C# surface this project modernizes — kept in the archived monorepo for consultation. |

New here? Read [`architecture/README.md`](architecture/README.md) first — it explains
what is kept from the original CAD model and what is replaced on the Go + Vulkan stack.

## License

Oblikovati is **dual-licensed** so vendors — including closed-source ones — can build
add-ins against a permissive contract while the application itself stays open
([ADR-0018](architecture/decisions/ADR-0018-apache-api-contract-module.md)):

- **[`Oblikovati.API`](https://github.com/Oblikovati/Oblikovati.API)** — the public
  automation contract (Go interfaces, value types, enums, JSON DTOs, typed client) —
  under the **Apache License 2.0**.
- **`Oblikovati`** (this repo) — the application (kernel, UI head, CLI) — under the
  [**GNU GPL v2**](LICENSE).
- **[`Oblikovati.AddIns`](https://github.com/Oblikovati/Oblikovati.AddIns)** — add-ins
  link only the Apache-2.0 contract and ship under their own (proprietary) license.

The split into three repos keeps the licenses cleanly separated (ADR-0018). Every
`.go` file carries an `SPDX-License-Identifier` header (`GPL-2.0-only` here). See the
[LICENSE](LICENSE) for the full statement.
