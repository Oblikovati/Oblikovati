## Code style

- Functions: 4-20 lines. Split if longer.
- Files: under 500 lines. Split by responsibility.
- One thing per function, one responsibility per module (SRP).
- Names: specific and unique. Avoid `data`, `handler`, `Manager`.
  Prefer names that return <5 grep hits in the codebase.
- Types: explicit. No `any`, no `Dict`, no untyped functions.
- No code duplication. Extract shared logic into a function/module.
- Early returns over nested ifs. Max 2 levels of indentation.
- Exception messages must include the offending value and expected shape.

## Comments

- Keep your own comments. Don't strip them on refactor — they carry
  intent and provenance.
- Write WHY, not WHAT. Skip `// increment counter` above `i++`.
- Docstrings on public functions: intent + one usage example.
- Reference issue numbers / commit SHAs when a line exists because
  of a specific bug or upstream constraint.

## Tests

- Tests run with a single command: `eg: go test`.
- Every new function gets a test. Bug fixes get a regression test.
- Mock external I/O (API, DB, filesystem) with named fake classes,
  not inline stubs.
- Tests must be F.I.R.S.T: fast, independent, repeatable,
  self-validating, timely.

## Dependencies

- Inject dependencies through constructor/parameter, not global/import.
- Wrap third-party libs behind a thin interface owned by this project.

## Structure

- Follow the framework's, sdk's language's convention.
- Prefer small focused modules over god files.
- Predictable paths: controller/model/view, src/lib/test, etc.

## Formatting

- Use the language default formatter (`cargo fmt`, `gofmt`, `prettier`,
  `black`, `rubocop -A`). Don't discuss style beyond that.

## Logging

- Structured JSON when logging for debugging / observability.
- Plain text only for user-facing CLI output.

## Directory Structure

**This repo is the GPL-v2 application** (`github.com/Oblikovati/oblikovati`): the
kernel, UI head, CLI, tests, and release pipeline live at the repo root. The
Apache-2.0 contract was split out into a sibling repo for licensing reasons
(ADR-0018); it is resolved for local dev via the `go.work` workspace over sibling
checkouts (not committed; CI checks out the sibling).

This repo's top-level layout:
- the Go application module at the root — `kernel/`, `model/`, `app/`, `command/`,
  `event/`, `persistence/`, `renderer/`, `scene/`, `addin/` (incl. `addin/router`,
  which serves `api/wire`), and `cmd/` (the `oblikovati` and `oblikovati-cli`
  binaries). It `require`s `github.com/Oblikovati/api` and implements its contracts.
- /head -> the cgo Vulkan + Dear ImGui windowed app, a separate submodule so the
  cgo build never touches the headless-tested core. It vendors the C ABI header at
  `head/internal/addinhost/include/oblikovati_addin.h` (the contract an add-in's
  shared library is compiled against) so it builds standalone.
- /test-utilities -> utilities and artifacts to help us test the code (e.g. blender projects).
- /architecture -> HOW we want to build; Documentation for architecture, ADRs.
- /experiments -> disposable experiments to validate things quickly before implementation (git-ignored).
- /implementation-plan -> WHAT we want to build; A roadmap with milestones, features and pbis.

The Autodesk Inventor C# API reference (Oblikovati.Contracts.CSharp) is intentionally
NOT in this repo — it lives, read-only, in the archived monorepo
(github.com/Oblikovati/Oblikovati.Contracts) for consultation.

Sibling repo (checked out alongside this one; tied together by `go.work`):
- ../Oblikovati.API -> the public API contract, a standalone Go module
  (`github.com/Oblikovati/api`), **Apache-2.0**. Four packages: `types` (enums, ids,
  value structs), `contract` (in-proc Go interfaces), `wire` (method-name constants
  + JSON DTOs), `client` (a `Transport` + typed client for add-ins). The source of
  truth for the API; it must NEVER import this application module (the dependency
  only flows the other way; CI enforces it). See ADR-0018.

Add-ins are separate shared libraries (their own repos, not this one). They link
**only** ../Oblikovati.API (Apache-2.0) — never this GPL module in their shipped
build — and reach the host over the C ABI (ADR-0016). Because the contract is
Apache-2.0, an add-in may be licensed however its author chooses, including
closed-source.

## Public API & implementation split (MANDATORY for all new work)

Adding to the public surface is always **two parts, in this order** (ADR-0018):

1. **Contract first, in ../Oblikovati.API** (Apache-2.0):
   - put enums / value types in `api/types` (define them ONCE there; this module
     aliases them with `type X = types.X` so existing call sites don't change);
   - put scalar Go interfaces in `api/contract`;
   - put method-name constants + JSON request/response DTOs in `api/wire`;
   - add a typed method group in `api/client` for any new wire method.
2. **Implementation here** (GPL-v2): build the behavior in `kernel/`, `model/`,
   `app/`, or `head/`; satisfy the `api/contract` interfaces with a compile-time
   assertion (`var _ contract.X = (*impl.X)(nil)`); wire the method into
   `addin/router` keyed on the `api/wire` constant.

Rules:
- Never re-declare a DTO or method-name string in this module or in an add-in —
  import it from `api/wire`.
- Never reach the host from an add-in with raw JSON — use `api/client`.
- Every new exported `.go` file carries an `SPDX-License-Identifier` header
  (Apache-2.0 in ../Oblikovati.API, GPL-2.0-only here); run
  `scripts/add-spdx-headers.py`.