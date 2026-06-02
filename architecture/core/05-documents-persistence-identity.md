# Core 05 — Documents, persistence & reference keys

*Modernizes M03 (documents, structured storage, references, reference keys,
attributes). Two big modernizations: a portable package format replacing OLE
structured storage, and reference keys as serializable Go values.*

## Documents & the document/content split

The split the parametric-cad skill (§1b) insists on is preserved: the **document**
is the file/identity/lifecycle/reference unit; the **component definition** is the
modeling content. This is what lets one part be referenced by many assemblies.

```go
package doc
type Document struct {
    id        ID
    typeID    TypeID         // part | assembly | drawing | presentation (stable)
    name      string
    dirty     bool
    content   compdef.Content// PartContent | AssemblyContent (the modeling data)
    refs      *RefGraph      // documents this references / is referenced by
    props     attr.PropertySet
}
type Workspace struct {     // replaces COM `Documents` collection + Application ownership
    open   Collection[*Document]
    active *Document
}
```

`Workspace` lives on the runtime mediator (core/00) — no global `Application`.

## Persistence — drop OLE structured storage

COM used OLE compound files (`.ipt`). That is a Windows-era, single-writer binary
storage format. We replace it with a **portable package container** that is
cross-platform and tool-friendly:

```
part.obk                       # a ZIP package (like .3mf/.docx/.usdz)
 ├── manifest.json             # version, TypeID of root, unit prefs, thumbnail ref
 ├── model/                    # the feature program, parameters, sketches (binary streams)
 │    ├── parameters.bin
 │    ├── features.bin
 │    └── sketches.bin
 ├── identity/keys.bin         # reference-key contexts (see below)
 ├── attributes.bin            # attribute sets / iProperties
 ├── thumbnail.png
 └── cache/                    # OPTIONAL derived data (tessellation), regenerable, may be omitted
```

Design choices:
- **Streams are columnar binary** via registered `Codec[T]`s keyed by stable
  `TypeID` (core/02). We serialize the **recipe** (parameters + feature definitions
  + sketches), not the evaluated B-rep — the geometry is a cache that recompute
  rebuilds (parametric-cad §0: the definition is the truth, geometry is a cache).
- **Atomic save**: write a temp package, fsync, rename — no half-written files
  (replaces COM compaction-on-save with a simpler invariant).
- **The package is inspectable** (it's a zip) — huge for debugging, diffing,
  and CI, versus an opaque OLE blob.
- **Migration**: `manifest.json` carries a schema version; an on-load migration
  pipeline upgrades older packages (replaces COM `OnMigrateDocument`).

## Reference keys — the hard problem, as Go values

This is the single most load-bearing mechanism (parametric-cad §7). A reference
key must re-resolve to "the same" face/edge after recompute destroys and recreates
the B-rep, and after save/reload. In Go it is a **serializable value**, not a
pointer:

```go
package identity
type RefKey struct {        // opaque to callers; serializable; persisted in identity/keys.bin
    ctx     ContextID
    payload []byte          // encodes the GENERATIVE LINEAGE (topo.Lineage, core/03)
}
type KeyManager struct{ ... }
func (m *KeyManager) Key(e topo.Entity) RefKey                     // mint from lineage
func (m *KeyManager) Resolve(k RefKey) (topo.Entity, bool)         // rebind; bool=false ⇒ LOST
func (m *KeyManager) NewContext() ContextID                        // versioned snapshot
func (m *KeyManager) Save(w io.Writer) ; Load(r io.Reader)         // persist contexts
```

The design is driven entirely by the kernel owning topology (ADR-0002): because
every `topo.Face` carries its `Lineage` ("end cap of feature F"), the key encodes
that derivation path, and after a rebuild the manager re-finds the entity whose
lineage matches — **topological naming, not addresses.**

**Binding may fail, and that is normal** (parametric-cad §7): `Resolve` returns
`false` when the referenced topology genuinely no longer exists. The contract: a
consumer with a lost key sets its **feature health to sick** and surfaces for
re-selection — never crashes. This is wired the same everywhere (features,
dimensions, mates) via core/06's health propagation.

> **Build this early.** The reference-key design is decided *before* the feature
> engine depends on selecting topology (iteration 2), not after — retrofitting it
> is the most expensive mistake in the field.

## Document references & file resolution

The reference graph (referenced / referencing / all-referenced) is preserved as a
Go graph; file resolution uses a **project/workspace search-path** model (replacing
COM `FileManager`/projects) — a portable config of library/workspace roots, no
registry, no OLE monikers. Broken references are flagged, never fatal.

## Attributes & properties

Unchanged in concept (parametric-cad §11): typed attribute sets attachable to any
entity (keyed by `RefKey` so they survive recompute) and document property sets
(iProperties). In Go: `attr.Set` with typed values, serialized to `attributes.bin`.
The `NameValueMap`-as-everything COM habit is replaced by **typed option structs**
in-proc; the only string→value maps left are at the gRPC seam (ADR-0003/0006).
