# Modeling 03 — Dress-up & pattern features

*Modernizes M09 (dress-up, holes/boss, patterns/mirror, modify/direct-edit). These
features consume **picked topology** — so they lean hardest on persistent identity
(parametric-cad §7, ADR-0010) and on the kernel's harder phases (ADR-0002).*

The defining property of this milestone: every feature here takes **reference-keyed
edge/face selections as inputs**. Reference-loss handling (a picked edge vanishes
after an upstream edit → feature goes sick, fixable by re-selection) is not an edge
case — it is the central behavior, and it works because of the lineage→reference-key
seam (modeling/01).

## Dress-up features

```go
type FilletDefinition struct {
    ConstantSets []ConstantRadiusSet  // each: Edges []Ref + Radius param.Quantity
    VariableSets []VariableRadiusSet  // edges + per-point radii
    Setbacks     []Setback
    FullRounds   []FullRoundSet       // three-face full rounds
}
type ConstantRadiusSet struct{ Edges []Ref; Radius param.Quantity }
```

| Feature | Inputs (reference-keyed) | Kernel op | Phase |
|---|---|---|---|
| **Fillet** | edge sets (constant/variable radius), setbacks, full-rounds | `ops.Fillet` | B (rolling-ball) / C |
| **Chamfer** | edges; distance \| distance-angle \| two-distance | `ops.Chamfer` | A (analytic) / C |
| **Shell** | faces to remove + thickness (uniform/per-face) | `ops.Shell` | C |
| **Draft** | neutral plane `Ref` + faces + pull direction + angle | `ops.Draft` | B/C |
| **Thread** | cylindrical face `Ref` + thread spec (cosmetic/modeled) | `ops.Thread` | A (cosmetic) |

- Edge/face inputs are `[]Ref`; resolved at recompute. A set whose edges are all
  lost → that set drops; all lost → feature sick (modeling/01).
- `FilletDefinition` reflects directly into the inspector (core/09): edge-set rows
  with a radius `Quantity` each, edit → `EditDefinition` command → async recompute.
- **Phase reality:** chamfer-on-analytic and cosmetic threads run in **Phase A** (no
  general blending needed); rolling-ball fillets need **Phase B**; shell and complex
  variable fillets need **Phase C** robust blending/offset. The *feature and its UI*
  exist from the start; the op returns "not yet supported" health until its phase
  lands (a `NotYetImplemented` health, not a missing button — core/01 convention).

## Hole & boss

```go
type HoleDefinition struct {
    Placement HolePlacement  // sealed: Linear | Concentric | OnPoint | Sketch(refs)
    Type      HoleType        // simple | counterbore | countersink | spotface
    Tap       *TapInfo        // nil = clearance; else thread spec
    Extent    Extent          // reuse the sketched-feature sealed extent (modeling/02)
}
```

Holes reuse the **sealed `Extent`** sum type from modeling/02 — consistency across
features. `TapInfo` carries thread data that later feeds hole tables and drawings
(iteration 4, M14). Placement is a sealed sum type (no nullable-field combos). Boss
features follow the same triangle.

## Patterns & mirror

Replicate a *source feature* parametrically, with per-element control.

```go
type RectangularPatternDefinition struct {
    Features  []Ref            // the feature(s) to replicate
    Dir1, Dir2 PatternAxis      // direction ref + count param + spacing param
    Compute   ComputeKind       // identical | adjust | orient
    Suppressed []int            // per-element suppression
}
```

- A pattern is itself a `Feature`; its **elements** are addressable for suppression/
  repositioning (the COM `FeaturePatternElement`). Editing the count parameter
  changes instances; recompute re-evaluates.
- **Mirror** reflects features/bodies across a plane `Ref`. **Sketch-driven** places
  instances at sketch points (a `Ref` to the sketch).
- Patterns multiply geometry cheaply: the pattern replays the source op N times in
  the engine (modeling/01); independent instances tessellate in parallel (core/08).
- *Assembly* component patterns (occurrences, not features) are iteration 3 (M11) —
  the same idea one level up, on occurrences + transforms (parametric-cad §5).

## Modify & direct-edit

```go
type CombineDefinition  struct{ Base Ref; Tools []Ref; Op OpKind }      // multi-body boolean
type MoveFaceDefinition struct{ Faces []Ref; Transform math.Mat4 }       // direct face edit
type DirectEditDefinition struct{ Targets []Ref; Action EditAction }     // push/pull/size/rotate/delete
```

| Feature | Inputs | Kernel op | Phase |
|---|---|---|---|
| **Combine / Split** | base + tool bodies / split tool `Ref` | `ops.Boolean` / `ops.Split` | C |
| **Move/Offset/Delete/Replace Face** | face `[]Ref` + transform/heal | `ops.FaceEdit` | C |
| **Thicken/Offset** | surface/face `Ref` + thickness | `ops.Thicken` | B/C |
| **Direct edit** | face/feature `[]Ref` + action | `ops.DirectEdit` | C |

These are the most kernel-demanding features (general booleans, face healing) and
mostly land in **Phase C**. The architecture is ready for them now — the engine,
reference-keyed inputs, commands, inspector, and async recompute all exist; they are
gated only by the pure-Go kernel maturing (ADR-0002), which is the whole project's
long pole and exactly why we phase it.

## Why identity carries this milestone

Every feature here is a stress test of reference keys. If the lineage→key→resolve
seam (core/03 → core/05 → modeling/01) is right, a fillet survives an upstream
parameter change that re-creates its edges, and *fails gracefully* (sick, re-
selectable) when an edge truly disappears. If it is wrong, dress-up features
constantly "lose" their edges on edits — the classic MCAD failure mode. This is the
concrete payoff of the parametric-cad skill's §7 insistence that identity be
designed before features depend on it — which ADR-0010 committed to before a single
feature in this doc was written.
