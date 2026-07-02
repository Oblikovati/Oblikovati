# Assembly 00 — Instancing & context proxies

*Modernizes M11 (assembly content, occurrences, proxies, patterns, BOM). The
prototype/flyweight model (parametric-cad §5) and the generic context proxy
(§5a / core/02) made concrete.*

## The flyweight: one definition, N occurrences

```go
package assembly
type Content struct {                 // the AssemblyComponentDefinition
    occurrences Collection[*Occurrence]
    constraints []Constraint          // (doc 01)
    joints      []Joint
    reps        Representations        // (doc 02)
    bom         *BOM
}
type Occurrence struct {
    def       compdef.Content          // ← SHARED (a part or subassembly's content)
    defRef    Ref                      // reference to the source document (core/05)
    transform math.Mat4                // THIS instance's placement
    grounded  bool
    suppressed bool
    parent    *Occurrence              // nesting
    subs      []*Occurrence            // sub-occurrences (for subassembly instances)
}
```

The non-negotiable property (parametric-cad §5): editing the **definition** updates
**every** occurrence; per-instance state (transform, grounded, suppressed, color
override) lives on the **occurrence**. Placing the same part twice creates two
`Occurrence`s sharing one `Content` — never a copy.

- **`def` is shared content**, reached through the document reference graph (core/05).
  A part open in its own window and placed in an assembly is *one*
  `PartComponentDefinition`.
- **Nesting → occurrence path.** A specific instance deep in subassemblies is
  identified not by a pointer but by its **`OccurrencePath`** (the chain of
  occurrences from the top assembly down) — two instances of the same screw differ
  only by path + transform. Paths are how selection, BOM, and proxies address
  instances.

```go
type OccurrencePath []*Occurrence      // top → … → leaf
func (p OccurrencePath) Transform() math.Mat4   // accumulated placement (product of transforms)
```

## Context proxies — the generic `Proxy[E]` (core/02) in action

A face belongs to a *definition* (part space). An assembly mate, a measurement, or a
dimension needs that face *as positioned in this occurrence* (assembly space). The
bridge is the **one generic proxy** decided in core/02 — assembly is its first user:

```go
// core/02
type Proxy[E Entity] struct { Native E; Context *OccurrenceContext }
type OccurrenceContext struct{ Path OccurrencePath }   // accumulated transform + identity

func CreateGeometryProxy[E Entity](occ OccurrencePath, native E) Proxy[E] {
    return Proxy[E]{Native: native, Context: &OccurrenceContext{Path: occ}}
}
```

- `proxy.Geometry()` applies `Context.Path.Transform()` to the native geometry →
  **assembly-space** geometry; `proxy.TypeID()` is the native's → **same identity**.
- The architectural win the skill demands (§5a) is preserved by the type system:
  `Proxy[*topo.Face]` is **not** assignable to `*topo.Face`, so part-space geometry
  can never be silently used where assembly-space is required. The compiler enforces
  the distinction the COM 275-`*Proxy`-class zoo enforced by hand-generation.
- **One implementation, all entity kinds** — `Proxy[*topo.Edge]`, `Proxy[*WorkPoint]`,
  `Proxy[*topo.Body]` are instantiations, not 275 hand-written types.

## Recompute & propagation (reuses ADR-0007/0010)

Assembly recompute has two jobs, both on the existing machinery:

1. **Propagate part edits up.** When a part's content changes (its own async
   recompute, ADR-0010), every assembly referencing it is dirtied via the document
   reference graph (core/05); affected occurrence geometry re-tessellates. Because
   the definition is *shared*, this is one recompute feeding N occurrences.
2. **Re-solve positions.** When a constraint/joint or a driving parameter changes,
   the assembly position solve (doc 01, ADR-0011) re-runs — async, cancellable,
   warm-started. Until it finishes, occurrences render at last-good transforms.

**Flexible subassemblies** expose a subassembly's internal DOF to the parent solve —
modeled as the subassembly contributing its unsolved DOF to the parent's
`solve.System` instead of being rigid. Just more variables; no special path.

## Component patterns, mirror, substitution

- **Occurrence patterns** (circular/rectangular/feature-based) replicate an
  occurrence with parametric count/spacing — like feature patterns (modeling/03) but
  one level up: each element is an `Occurrence` sharing the same `def`, with a
  computed transform. Per-element suppression via `OccurrencePatternElement`.
- **Mirror components** produce correctly-handed instances (a true mirror needs a
  mirrored *definition*, a derived part — modeling/02; a symmetric part can reuse the
  definition with a mirror transform). The definition decides.
- **Substitution** swaps an occurrence's `def` for a simplified representation
  (shrinkwrap — shipped as `compdef.shrinkwrap_lod`, M11-F06) — the occurrence keeps
  its path/transform, changes what it points at.

## GPU instancing falls out (core/08, realtime-3d §3, §6)

Because occurrences share a definition, they share its **tessellated mesh** in the
renderer cache (keyed by body reference key). N occurrences = N scene entities, each
with the occurrence's transform, all referencing one mesh handle → a single
**instanced draw** (draw-call-as-data carries per-instance transforms). A 10,000-bolt
assembly is one mesh + 10,000 transforms, not 10,000 meshes. The flyweight in the
model and instancing on the GPU are the *same idea* at two layers — the realtime-3d
and parametric-cad skills meeting exactly here.

## BOM — derived from occurrence structure

```go
type BOM struct{ views []*BOMView }   // structured | parts-only
type BOMRow struct{ def compdef.Content; structure BOMStructure; qty Quantity; item int }
```

The BOM is **derived** from the occurrence tree: group occurrences by shared
definition → quantities; classify by `BOMStructure` (normal/phantom/reference/
purchased); assign stable item numbers. Phantom subassemblies collapse into their
parent. It is a *query* over the assembly (rebuilt on structure change), feeding
parts lists and balloons in drawings (iteration 4, M14). Custom columns source from
iProperties (core/05).

## Net mapping from COM

| COM | Here |
|---|---|
| `ComponentOccurrence` + `Definition` + `Transformation` | `Occurrence{def shared, transform}` flyweight |
| `OccurrencePath` / `SubOccurrences` | `OccurrencePath` slice + `subs` |
| `CreateGeometryProxy` + 275 `*Proxy` types | one generic `Proxy[E]` (core/02) |
| `CircularOccurrencePattern` etc. | occurrence pattern = feature-pattern one level up |
| `BOM`/`BOMView`/`BOMRow` | derived query over occurrence structure |
