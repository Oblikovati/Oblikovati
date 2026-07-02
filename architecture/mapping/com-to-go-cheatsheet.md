# COM/C# → Go/Vulkan idiom cheatsheet

A quick-reference translation table from the original `Oblikovati.Contracts` (COM-
mirroring) idioms to the modernized Go architecture. Use it when porting any of the
2200+ interfaces. Deep rationale is in the ADRs and `core/` docs.

## Object model & types

| COM / C# | Go | Notes / ref |
|---|---|---|
| `ObjectTypeEnum Type { get; }` on every object | Go static type; `TypeID uint32` **only** for storage/RPC | [core/02](../core/02-object-model-identity.md), ADR-0006 |
| `object` / `VARIANT` param or return | concrete type + generics; `any`/proto `oneof` only at RPC seam | core/02 |
| `_X : X` dual interface (versioning) | add methods/fields freely; protobuf field numbers at seam | core/02 |
| `obj.Parent`, `obj.Application` | `*Runtime` mediator (passed); explicit `parent` only in the model tree | [core/00](../core/00-runtime-and-frame-loop.md) |
| `IEnumerable` collection, `Item[1]` (1-based) | `Collection[T]`, 0-based, `range` | core/02 |
| `*Enumerator` (105 types) | `iter.Seq[T]` / `[]T` snapshot | core/02 |
| `NameValueMap` options bag | typed option struct in-proc; `map<string,Value>` at seam | core/05, ADR-0006 |
| `*Proxy` (275 types) | one generic `Proxy[E Entity]` | core/02, parametric-cad §5a |
| `HRESULT` / COM error | Go `error` (typed); modeling fail = `Health`, not error | ADR-0001 |

## Geometry & kernel

| COM / C# | Go | Notes |
|---|---|---|
| `TransientGeometry.CreatePoint(...)` factory | `geom.Point{X,Y,Z}` value struct (no factory) | [core/03](../core/03-geometry-kernel.md) |
| `TransientObjects.CreateObjectCollection` | `Collection[T]` / `[]T` | core/02 |
| native kernel via Coral marshaling | pure-Go `kernel/` (no cgo) | ADR-0002 |
| `SurfaceBody`/`Face`/`Edge` (COM topo) | `topo.Body`/`Face`/`Edge` with `Lineage` | core/03 |
| `FaceEvaluator.GetNormal(...)` | `srf.NormalAt(u,v)` method | core/03 |
| `ref byte[] ReferenceKey` | `identity.RefKey` (serializable value) | [core/05](../core/05-documents-persistence-identity.md) |
| `ReferenceKeyManager.BindKeyToObject` | `identity.RefKey` tiered bind/recover: exact lineage match, else `identity.RecoverLost` (ancestral → geometric); `MatchNone` ⇒ lost (`model/identity/refkey.go`) | core/05, parametric-cad §7 |

## Parameters

| COM / C# | Go | Notes |
|---|---|---|
| `Parameter.Expression` (string) | `param.Expr` (parsed AST, refs by ID) | [core/04](../core/04-parameters-expressions.md) |
| `object Value` + `string Units` | `param.Quantity{Value float64; Unit}` (database units) | core/04 |
| `ModelValue` (post-tolerance) | `modelValue float64` field | core/04 |
| `Dependents` / `DrivenBy` | `param.Graph` edges + `DirtyClosure` | core/04 |
| `ExpressionList` | `[]Expr` / `iter.Seq[Expr]` | core/04 |

## Documents, persistence, identity

| COM / C# | Go | Notes |
|---|---|---|
| `Documents` collection + `Application` | `doc.Workspace` on the runtime | core/05, core/00 |
| `Document` vs `ComponentDefinition` split | `doc.Document` vs `compdef.Content` (kept!) | core/05, parametric-cad §1b |
| OLE structured storage (`.ipt`) | zip package (`.obk`) + columnar binary streams | core/05 |
| `DataIO` streams | registered `Codec[T]` keyed by `TypeID` | core/05 |
| compaction / `OnMigrateDocument` | atomic save (temp+rename); manifest-versioned migration | core/05 |
| `FileManager` / projects | portable search-path config | core/05 |
| `AttributeSet`/`Attribute` | `attr.Set` typed values, keyed by `RefKey` | core/05, parametric-cad §11 |

## Transactions, undo, events

| COM / C# | Go | Notes |
|---|---|---|
| `TransactionManager.StartTransaction` | `command.History.Do(doc, cmd)` | [core/06](../core/06-transactions-and-events.md), realtime-3d §11 |
| `Transaction` (nested/global/merge) | `Command` / composite `Batch` command | core/06 |
| `CheckPoint` / `GoToCheckPoint` | history length marker / undo-to-length | core/06 |
| `XEventsObject` + `XEventsSink_Event` | `event.Subscribe[E](bus, phase, handler)` | core/06 |
| `EventTimingEnum BeforeOrAfter` | `event.Phase` (`Before`/`After`) | core/06 |
| `out HandlingCodeEnum` (veto) | handler returns `Outcome` (`Continue`/`Veto(reason)`) | core/06 |
| per-event delegate (316 types) | one generic `Handler[E]`; events are structs | core/06 |
| `ChangeManager` / `ChangeProcessor` | subscribe + issue commands (composed, not a framework) | core/06 |

## UI, commands, interaction, add-ins

| COM / C# | Go | Notes |
|---|---|---|
| `ApplicationAddInServer` (in-proc COM) | in-proc registry (1st-party) + gRPC process (3rd-party) | [core/07](../core/07-extensibility.md), ADR-0003 |
| `CommandManager` / `ControlDefinitions` | `registry.Commands` + `Command` interface | core/07 |
| `ButtonDefinition`/`ComboBoxDefinition`… | one `Command`; surface decided by UI | core/07 |
| `CommandBars` / Ribbon (COM UI) | ImGui ribbon generated from `registry.Commands` | [core/09](../core/09-ui-imgui.md) |
| `BrowserPane` / `BrowserNode` (retained) | ImGui tree built from live document each frame | core/09 |
| `DockableWindows` | ImGui docking | core/09, ADR-0004 |
| `Environment` / `EnvironmentManager` | registered `Workspace` (Sketch/Part/Assembly/Drawing) | core/07 |
| `SelectSet` | `*Selection` on runtime (`[]Entity`, evented) | core/09 |
| `HitTestManager` / `SelectionFilterEnum` | ID-buffer pick pass + filter | core/08, core/09 |
| `ClientGraphics` / `InteractionGraphics` | overlay pass; add-ins via `ClientGfx.Draw` | core/08, core/07 |
| reflection-driven property panels | reflect `Definition` struct + field tags → ImGui | core/09, realtime-3d §10 |

## Cross-cutting principle

When porting an interface, ask: *which of COM's three overloaded jobs is this
member doing?* — **behavior** (→ Go static types/interfaces), **identity/storage**
(→ stable `TypeID`/`RefKey`), or **boundary-crossing** (→ gRPC, and only there do
variants/maps survive). Most COM ceremony was boundary-crossing for a boundary we
deleted; it simply vanishes.
