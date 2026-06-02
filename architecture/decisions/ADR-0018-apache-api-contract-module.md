# ADR-0018 — Apache-2.0 `/api` contract module extracted from the GPL `/source`

**Status:** accepted (user decision, 2026-06) · **Amends:**
[ADR-0006](ADR-0006-no-com-object-model.md) (the public surface is now defined once
in `/api`, not "twice"), [ADR-0003](ADR-0003-extensibility-hybrid-rpc.md) /
[ADR-0016](ADR-0016-shared-library-addins-mcp-bridge.md) (the JSON method contract is
now the typed, Apache-licensed `api/wire` + `api/client`).

## Context

The application (kernel + UI head + CLI) ships under **GPL-2.0** (`/source`). Third
parties — possibly closed-source vendors — must be able to build add-ins. The public
automation API was **baked into** `/source`: the de-facto contract was the router's
JSON method strings plus request/response DTOs declared *inline* in the GPL router
package, and the MCP bridge re-declared the same DTOs. There was no Apache-licensed
artifact a closed add-in could compile against, and "the surface is defined twice"
(ADR-0006) had become literally true.

## Decision

Extract the public API into its own **Apache-2.0 Go module**,
`github.com/Oblikovati/api` (`/api`), as the single source of truth that both
`/source` and add-ins depend on. The dependency only ever flows **toward** `/api`;
`/api` never imports `/source` (CI-enforced).

The module is layered:

| Package | Holds | Who satisfies / consumes it |
|---|---|---|
| `api/types` | enums, stable ids, value/option structs — pure data | enum *definitions* (DocumentType, ParameterKind); `/source` aliases them |
| `api/contract` | in-process Go **interfaces** (Document, Parameter, …) | `/source` types satisfy them via compile-time assertions |
| `api/wire` | method-name constants + JSON request/response DTOs | the host router marshals into them; the wire boundary uses them |
| `api/client` | a `Transport` interface + a typed client over it | out-of-runtime add-ins call the host through it |

## Why this shape (the two consumption paths)

A Go c-shared add-in runs its **own Go runtime** (ADR-0016): a live Go interface
value cannot cross the C-ABI boundary. So the contract serves two audiences with two
mechanisms, deliberately:

- **In-process / first-party** code uses `api/contract` interfaces and `api/types`
  directly (one runtime). The compile-time assertions in `/source`
  (`var _ contract.Document = (*doc.Document)(nil)`) keep the published interface and
  the implementation from drifting.
- **Out-of-runtime / third-party** add-ins use `api/wire` + `api/client` over a
  transport (today the C-ABI `ObkHostCall`; tomorrow gRPC or a socket). They link
  **only** the Apache `/api` module, never the GPL `/source`, so a closed-source
  add-in stays decoupled from GPL both legally (separate module) and at the ABI (the
  only thing crossing the boundary is JSON, per ADR-0016).

`api/types` is how `/api` becomes the source of truth for enums without churning the
implementation: the enum is defined once in `api/types`, and `/source` keeps its
historical spelling via a transparent **type alias** (`type DocumentType =
types.DocumentType`) — existing call sites are unaffected.

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

- **Licensing is now explicit and enforced.** `api/LICENSE` = Apache-2.0,
  `source/LICENSE` = GPL-2.0, SPDX headers on every `.go` file
  (`scripts/add-spdx-headers.py`), and CI fails if `/api` ever imports `/source`.
- **No more duplicated DTOs.** The router and the MCP bridge share `api/wire`; the
  bridge's host-call abstraction is `api/client.Transport`.
- The multi-module repo grows one module (`/api`), wired via `replace` directives
  like `/source/head` already is.
- `gofmt`, `go vet`, `go test`, and a dependency-boundary check run against `/api`
  in CI alongside `/source`.

## Shape / layering

```
api/  (Apache-2.0, github.com/Oblikovati/api)
  types/  contract/  wire/  client/
source/  (GPL-2.0)  requires + implements api; serves api/wire via addin/router
add-in/  (vendor-licensed)  imports only api/{types,wire,client}
```
