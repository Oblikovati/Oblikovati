# Core 07 — Extensibility: registries + gRPC add-ins

*Modernizes M05-F01/F02 (COM add-in framework, command framework). Implements
ADR-0003. Applies realtime-3d skill §9 (self-registration) and §12 (dogfood).*

Two mechanisms by audience (ADR-0003): **in-process registries** for first-party
features, **out-of-process gRPC** for third-party add-ins.

## In-process self-registration (first-party)

The realtime-3d skill §9 pattern: feature packages register themselves at `init()`;
the host blank-imports them for the side effect. No central switch statement.

```go
// registry/registry.go
var Features  = NewRegistry[FeatureKind]()
var Workspaces= NewRegistry[Workspace]()    // Sketch, Part, Assembly, Drawing modes
var Commands  = NewRegistry[Command]()
var Translators = NewRegistry[Translator]() // STEP, DWG, STL… (M17)
var Inspectors  = NewRegistry[Inspector]()  // property-panel editors (reflection, core/09)

// model/feature/extrude/extrude.go
func init() { registry.Features.Register(extrudeKind) }   // ← adding extrude = this + one import
```

```go
// cmd/oblikovati/features.go — the host imports for side effects only
import (
    _ "oblikovati.org/model/feature/extrude"
    _ "oblikovati.org/model/feature/revolve"
    _ "oblikovati.org/model/feature/fillet"
    _ "oblikovati.org/workspace/sketch"
    _ "oblikovati.org/workspace/assembly"
)
```

Each registered thing implements a small **lifecycle interface** (identity +
`Init/Shutdown/Open/Close/Focus/Blur/Update` as applicable — realtime-3d §9). This
is how whole **workspaces** (Sketch, Part, Assembly, Drawing — the COM
"environments") plug in without touching a core dispatcher. Adding a feature type
or a translator is "add a package + one blank import," never editing a registry by
hand.

### Commands (the COM `ControlDefinition`/`CommandManager` replacement)

A command is a registered object with metadata + an interactive lifecycle, decoupled
from how it is surfaced (the ImGui ribbon reads the registry to build buttons —
core/09):

```go
type Command interface {
    Meta() CommandMeta             // id, label, icon, category, hotkey, enable-predicate
    Run(ctx *interaction.Context)  // drive selection/manipulators; commit via command.History
}
```

No COM `ButtonDefinition`/`ComboBoxDefinition` zoo — one interface; the *surface*
(button vs menu vs hotkey) is a UI concern, not baked into the definition.

#### Ribbon placement (the COM `Ribbons`/`RibbonTab`/`RibbonPanel` replacement)

The host has one ribbon per document type plus ZeroDoc, switched by the active
document (Inventor's RibbonUI model — `Inventor-API/files/RibbonUI_Overview.md`,
core/09). A command declares **where** it surfaces: a `RibbonKey` (which of the seven
ribbons — `ZeroDoc, Part, Assembly, Drawing, Presentation, iFeatures,
UnknownDocument`, default `Part`), a tab and panel within it, and an `Environment`
(base, or a contextual one like `sketch`). `BuildRibbon` then emits exactly the active
document's ribbon, with contextual tabs present only in their environment.

Add-ins manage the ribbon through the same contract (no special path):

- **Discover** — `ribbon.list` (`client.Ribbon().List()`) returns the active ribbon's
  `key → tabs → panels → controls`, so an add-in can find the internal names to insert
  next to — Inventor's "list the contents of the ribbon".
- **Place** — `commands.create` (`client.Commands().Create`) carries `Ribbon`,
  `Tab`, `Category`, and `Environment`, so a button lands on, say, the Draw panel of the
  Sketch tab of the Part ribbon. An unknown ribbon name is rejected, not silently
  defaulted.

`RibbonKey` and `Environment` are defined once in `api/types` (Apache-2.0) and aliased
in `/source`, like every other public enum (ADR-0018).

## Out-of-process add-ins (third-party) — gRPC

> **As built (2026-06, [ADR-0016](../decisions/ADR-0016-shared-library-addins-mcp-bridge.md)).**
> The first add-in (`oblikovati-mcp-bridge`) ships the third-party mechanism
> **in-process** as a C-shared library (`.so`/`.dll`) loaded over a small **C ABI**,
> exposing the API to LLM clients over **MCP**. The boundary carries an in-process
> **JSON method contract** (`commands.*`, `documents.*`, `parameters.*`, `model.*`,
> `sketch.*`, `features.*`) implemented by `source/addin/router`; the cgo loader +
> host-call shim live in `source/head/internal/addinhost`. That contract is
> transport-agnostic, so the gRPC services below remain the planned *future*
> transport behind the same surface (deferred, not abandoned). The rest of this
> section describes that target gRPC design.
>
> Since [ADR-0018](../decisions/ADR-0018-apache-api-contract-module.md) the contract
> is typed and Apache-2.0: the method names + DTOs live in `api/wire` and add-ins
> call through `api/client` (a `Transport` the C-ABI `ObkHostCall` backs), so a
> closed-source add-in links only `/api`, never the GPL `/source`.

Third-party add-ins are **separate processes** speaking the `api/` gRPC contract
(ADR-0003) — the modern successor to the COM type library and the
`Oblikovati.Contracts` surface.

```
addins/host.go         # discovers, launches, supervises add-in processes
addins/registry.json   # installed add-ins (manifest: exec, capabilities, permissions)
api/*.proto            # the public contract: Documents, Model, Edit, Events, Selection, ClientGfx
api/server.go          # serves the contract against the live Runtime (in-proc adapter)
```

Lifecycle (replaces `ApplicationAddInServer.Activate/Deactivate`):
- **Discovery**: scan add-in manifests; each declares an executable, capabilities,
  and requested permissions.
- **Launch & supervise**: the host starts each add-in as a child process, holds a
  gRPC connection, and **restarts/disables on crash** — the add-in cannot take
  down the host (the headline win over in-process COM).
- **Permissioned surface**: an add-in only sees the gRPC services its manifest was
  granted — a capability model COM never had.
- **Heartbeat & deadlines**: unresponsive add-ins are detected; vetoes time out
  (core/06).

### What an add-in can do (the contract surface)

| COM add-in capability | gRPC service |
|---|---|
| read/walk the model | `Model` (features, parameters, tessellation, selection sets) |
| create/edit features | `Edit` (Begin → AddFeature(Definition) → Commit), command-backed |
| subscribe to events / veto | `Events.Subscribe` stream + paired allow/veto reply |
| draw overlay (client graphics) | `ClientGfx.Draw(OverlayGraphics)` |
| add UI (ribbon button/panel) | `UI` service: register command metadata; host renders it in ImGui |
| open/save/translate | `Documents`, `Translators` |

The add-in holds **stable IDs / reference keys** across the wire (core/02/05), never
live pointers — so the boundary stays clean and the host stays in control of the
model.

## Dogfooding (realtime-3d §12)

First-party features use the **in-proc registries** for speed, but the gRPC `Edit`/
`Model` services are exercised in integration tests against real documents — if our
own automation can build a part end-to-end over gRPC, the public API is real and
complete (no "internal-only backdoors"). This is the validation that the contract
is sufficient, the same way the original `Oblikovati.Contracts` aspired to mirror a
complete automation surface.
