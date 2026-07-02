# ADR-0018 — Apache-2.0 API contract module extracted from the GPL application

**Status:** accepted (user decision, 2026-06) · **Amends:**
[ADR-0006](ADR-0006-no-com-object-model.md) (the public surface is now defined once
in the API module, not "twice"), [ADR-0003](ADR-0003-extensibility-hybrid-rpc.md) /
[ADR-0016](ADR-0016-shared-library-addins-mcp-bridge.md) (the JSON method contract is
now the typed, Apache-licensed `api/wire` + `api/client`).

> **Editor's note (2026-07, #1661 / M40 audit D4).** Because CLAUDE.md cites this
> ADR as *current* guidance for the public-API/implementation split, its path
> references were updated in place after the repo-root migration: the GPL
> application (written `/source` at decision time) lives directly at the
> repository root, and the Apache-2.0 `oblikovati.org/api` module lives in the
> sibling `Oblikovati.API` repository (still written `api/…` package paths
> below). The decision itself is unchanged.

## Context

The application (kernel + UI head + CLI) ships under **GPL-2.0** at the repo
root. Third
parties — possibly closed-source vendors — must be able to build add-ins. The public
automation API was **baked into** the application module: the de-facto contract was the router's
JSON method strings plus request/response DTOs declared *inline* in the GPL router
package, and the MCP bridge re-declared the same DTOs. There was no Apache-licensed
artifact a closed add-in could compile against, and "the surface is defined twice"
(ADR-0006) had become literally true.

## Decision

Extract the public API into its own **Apache-2.0 Go module**,
`oblikovati.org/api` (the sibling `Oblikovati.API` repo), as the single source of
truth that both the application and add-ins depend on. The dependency only ever
flows **toward** the API module; it never imports the application (CI-enforced).

The module is layered:

| Package | Holds | Who satisfies / consumes it |
|---|---|---|
| `api/types` | enums, stable ids, value/option structs — pure data | enum *definitions* (DocumentType, ParameterKind); the application aliases them |
| `api/contract` | in-process Go **interfaces** (Document, Parameter, …) | application types satisfy them via compile-time assertions |
| `api/wire` | method-name constants + JSON request/response DTOs | the host router marshals into them; the wire boundary uses them |
| `api/client` | a `Transport` interface + a typed client over it | out-of-runtime add-ins call the host through it |

## Why this shape (the two consumption paths)

A Go c-shared add-in runs its **own Go runtime** (ADR-0016): a live Go interface
value cannot cross the C-ABI boundary. So the contract serves two audiences with two
mechanisms, deliberately:

- **In-process / first-party** code uses `api/contract` interfaces and `api/types`
  directly (one runtime). The compile-time assertions in the application
  (`var _ contract.Document = (*doc.Document)(nil)`) keep the published interface and
  the implementation from drifting.
- **Out-of-runtime / third-party** add-ins use `api/wire` + `api/client` over a
  transport (today the C-ABI `ObkHostCall`; tomorrow gRPC or a socket). They link
  **only** the Apache API module, never the GPL application, so a closed-source
  add-in stays decoupled from GPL both legally (separate module) and at the ABI (the
  only thing crossing the boundary is JSON, per ADR-0016).

`api/types` is how the API module becomes the source of truth for enums without
churning the implementation: the enum is defined once in `api/types`, and the
application keeps its historical spelling via a transparent **type alias**
(`type DocumentType = types.DocumentType`) — existing call sites are unaffected.

## Scope (this pass) and what is deferred

Extracted: the surface the app exposes today — the router method contract
(`documents.*`, `parameters.*`, `model.*`, `sketch.*`, `features.*`, `commands.*`)
with its DTOs, the two enums on that surface (DocumentType, ParameterKind), and
`Document` / `Parameter` contract interfaces.

Deferred (grow milestone-by-milestone, not a rewrite of the 2200+ C# interfaces):
- **Collection / navigation interfaces** in `api/contract`. Go interfaces are
  invariant in return position, so `Document.Parameters() contract.Parameters`
  is not satisfied by a concrete method returning `*param.Parameters`. Expanding the
  contract past leaf accessors needs generics or adapter wrappers — a deliberate
  later design step. Out-of-proc add-ins navigate via `api/wire` meanwhile.
- Moving richer value types (Quantity, Unit, Health, reference keys) into
  `api/types`.

## Consequences

- **Licensing is now explicit and enforced.** The API module's `LICENSE` =
  Apache-2.0, this repo's `LICENSE` = GPL-2.0, SPDX headers on every `.go` file
  (`scripts/add-spdx-headers.py`), and CI fails if the API module ever imports
  the application.
- **No more duplicated DTOs.** The router and the MCP bridge share `api/wire`; the
  bridge's host-call abstraction is `api/client.Transport`.
- The API becomes its own module (`oblikovati.org/api`), resolved for local dev
  via the `go.work` workspace over sibling checkouts (like `head/` already is;
  CI checks out the sibling and injects the same mapping).
- `gofmt`, `go vet`, `go test`, and a dependency-boundary check run against the
  API module in CI alongside the application.

## Shape / layering

```
../Oblikovati.API/  (Apache-2.0, oblikovati.org/api — sibling repo)
  types/  contract/  wire/  client/
this repo root  (GPL-2.0)  requires + implements api; serves api/wire via addin/router
add-in repos  (vendor-licensed)  import only api/{types,wire,client}
```

## Addendum (2026-06): transient geometry — in-process values, wire DTO encoding

M01-F05 (#602) put the transient-geometry vocabulary on the public surface. These
values are ownerless, immutable, and created in huge volume (every point of every
stroke of every preview), so round-tripping construction and evaluation over the
C ABI would be the wrong transport. The split:

- **Value types live IN `api/types`, with their math** (`Point`/`Point2d`,
  `Vector`/`Vector2d`, `UnitVector`/`UnitVector2d`, `Matrix`/`Matrix2d`, the
  boxes, the B-spline definition recipes). They are pure data + pure functions —
  Apache-2.0, no host state — so an add-in computes locally at zero wire cost.
  This is the one place the contract module carries *implementation*, justified
  exactly because the implementation is closed-form math with no host coupling.
- **Wire encoding is the same types.** Their JSON form is fixed-length arrays
  (`[x,y,z]`, 9/16 matrix cells), byte-compatible with the ad-hoc
  `[3]float64`/`[16]float64` fields that predated them; payloads that carry
  geometry use them directly (camera, lighting, gizmos, anchors).
- **Curves/surfaces are contracts, not values.** Construction validates and
  evaluation runs against the kernel, so `api/contract` defines the interfaces
  (umbrella `Curve`/`Curve2d`/`Surface`, per-kind interfaces, the single
  `TransientGeometry` factory) and `kernel/geomapi` implements them as
  thin adapters over `kernel/geom` — the kernel itself stays free of API
  conversions. Kind discriminators are frozen to the reference enum values.
- The kernel keeps its own `math` package (`Scalar`-based) as the private
  computation vocabulary; `geomapi` converts at the boundary. Aliasing the two
  was rejected: it would couple every kernel file to the contract module for no
  behavioral gain.
