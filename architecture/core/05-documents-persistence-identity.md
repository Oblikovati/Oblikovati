# Core 05 — Documents, persistence & reference keys

*Modernizes M03 (documents, structured storage, references, reference keys,
attributes). Two big modernizations: a git-friendly single-file document format
replacing OLE structured storage (ADR-0020), and reference keys as serializable
Go values with tiered recovery (`model/identity`).*

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
    content   Content        // PartComponentDefinition | AssemblyComponentDefinition (the modeling data)
    refs      *RefGraph      // documents this references / is referenced by
    props     attr.PropertySet
}
type Workspace struct {     // replaces COM `Documents` collection + Application ownership
    open   Collection[*Document]
    active *Document
}
```

The live containers are `compdef.PartComponentDefinition` and
`compdef.AssemblyComponentDefinition` (`model/compdef`); `doc.Content` is the
seam that lets `model/doc` hold either without importing the heavy model
packages. `Workspace` lives on the runtime mediator (core/00) — no global
`Application`.

## Persistence — drop OLE structured storage

COM used OLE compound files (`.ipt`). That is a Windows-era, single-writer binary
storage format. We replace it with a **single readable YAML file**
([ADR-0020](../decisions/ADR-0020-yaml-git-friendly-document-format.md)) that is
cross-platform, tool-friendly, and diffs line-by-line in git:

```yaml
# part.obk — one YAML document (persistence/yamlcodec)
schemaVersion: 2            # migration gate (persistence.CurrentSchemaVersion)
documentType: 1             # stable TypeID of the root document kind
displayName: part
identity: { ... }           # file identity block (doc GUID, versions)
references: [ ... ]         # as-saved file-to-file reference records
model:                      # THE RECIPE — native nested YAML, not a quoted blob
  parameters: [ ... ]
  sketches: [ ... ]
  features: [ ... ]
resources: { ... }          # embedded imported files, base64, keyed by UUID (ADR-0031)
data: { ... }               # add-in/attribute scratch sections, base64
```

Design choices:
- **We serialize the recipe** (parameters + feature definitions + sketches), not
  the evaluated B-rep — the geometry is a cache that recompute rebuilds
  (parametric-cad §0: the definition is the truth, geometry is a cache).
- **Atomic save**: write a sibling temp file, fsync, rename (`persistence/io.go`)
  — no half-written files (replaces COM compaction-on-save with a simpler
  invariant).
- **The file is inspectable** (it's YAML) — huge for debugging, diffing, and CI,
  versus an opaque OLE blob. Only `persistence/yamlcodec` touches the YAML
  library (dependency rule).
- **Migration**: the top-level `schemaVersion` gates an on-load migration
  pipeline (`persistence/migration.go`) that upgrades older files step by step
  (replaces COM `OnMigrateDocument`).

## Reference keys — the hard problem, as Go values

This is the single most load-bearing mechanism (parametric-cad §7). A reference
key must re-resolve to "the same" face/edge after recompute destroys and recreates
the B-rep, and after save/reload. In Go it is a **serializable value**, not a
pointer — `identity.RefKey` (`model/identity/refkey.go`):

```go
package identity
type RefKey struct {        // opaque to callers; a value type — storable in any recipe
    ctx     ContextID       // topology context the key was minted in
    kind    EntityKind      // face | edge | vertex | ...
    payload []byte          // the entity's LineageKey at mint time (topo.Lineage, core/03)
    parent  ancestryHint    // session-only: parent lineage, for ancestral recovery
    anchor  anchorHint      // session-only: representative point, for geometric tie-break
}
func (k RefKey) Encode() []byte                    // persistable bytes (round-trip stable)
func DecodeKey(data []byte) (RefKey, error)
func RecoverLost(kind EntityKind, parentKey []byte,
    anchor *math.Point3, ents []Entity) (Entity, MatchType) // the fallback tiers
```

The design is driven entirely by the kernel owning topology (ADR-0002): because
every `topo.Face` carries its `Lineage` ("end cap of feature F"), the key encodes
that derivation path, and after a rebuild the binder re-finds the entity whose
lineage matches — **topological naming, not addresses.** Keys live *inside* the
recipe (feature and sketch definitions store the encoded bytes); there is no
separate key store or stream on disk.

**Binding degrades through tiers, and losing is normal.** Resolution tries
**exact → ancestral → geometric** and reports which tier matched via
`identity.MatchType`:

1. **Exact** — a single entity with the key's kind and lineage exists
   (`MatchExact`).
2. **Ancestral** — the exact entity is gone, but exactly one surviving sibling
   shares the key's parent lineage (`MatchAncestral`, auto-healed, flagged).
3. **Geometric** — several siblings survive; the one nearest the key's mint-time
   anchor point wins (`MatchGeometric`, auto-healed, flagged).
4. **None** — nothing matches (`MatchNone`): the reference is **lost**, a
   legitimate, non-fatal outcome.

`MatchType.IsFallback()` is the binder's signal to resolve the reference to
**Warning** health (auto-healed, review advised) instead of fully healthy; a lost
key sets **feature health to sick** and surfaces for re-selection — never
crashes. This is wired the same everywhere (features, dimensions, mates) via
core/06's health propagation. The fallback hints are an in-memory session
optimisation: they are not serialized, so a key reloaded from disk degrades to
exact-only binding until re-minted (M31-F06; see `model/identity/binding.go` and
`recover.go`; ADR-0043 generalizes the lineage/provenance naming the tiers match
against).

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
(iProperties). In Go: `attr.Set` with typed values, serialized into the
document's attribute block. The `NameValueMap`-as-everything COM habit is
replaced by **typed option structs** in-proc; the only string→value maps left are
at the wire seam (ADR-0003/0006).
