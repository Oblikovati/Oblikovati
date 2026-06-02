# Modeling 02 — Sketched & work features

*Modernizes M08-F02 (work features), F03 (sketched features), F04 (derived). Each
feature is the Definition→Add→Feature triangle (modeling/01) calling kernel ops
(core/03). Availability tracks the kernel phase (ADR-0002).*

## Work features (datums) — parametric construction geometry

Datum planes/axes/points defined **by relationship**, recomputing like features and
valid as inputs wherever no model face yet exists (parametric-cad §10).

```go
type WorkPlaneDefinition struct {
    method WorkPlaneMethod   // offsetFromPlane | throughThreePoints | tangentToFace | byLineAndPoint | ...
    refs   []Ref             // the geometry it is defined against (reference-keyed)
    offset param.Quantity
}
```

- A datum is a `Feature` (history node), not UI scaffolding — it recomputes, has
  health, and can be suppressed.
- An offset work-plane moves when its driving parameter changes (parametric); a
  three-point plane moves when its referenced points move (via reference keys).
- `WorkAxis`, `WorkPoint`, and `UserCoordinateSystem` follow the same pattern.
  Phase-A geometry suffices for all of them — build these early; sketches and
  features depend on them.

## Sketched features — extrude as the reference implementation

Extrude is built first and exemplary; every other sketched feature copies its shape
(parametric-cad §3 note). The full triangle:

```go
// the recipe (POD + Quantity + Ref) — also the inspector source & gRPC payload
type ExtrudeDefinition struct {
    Profile   Ref          `label:"Profile" pick:"profile"`
    Operation OpKind        `label:"Operation" enum:"join,cut,intersect,newbody,surface"`
    Extent    Extent        // sealed interface — see below
    Taper     param.Quantity `label:"Taper" unit:"angle"`
    Direction ExtentDir      `label:"Direction" enum:"positive,negative,symmetric"`
}
func (ExtrudeDefinition) Kind() feature.Kind { return KindExtrude }

// extents are a sealed sum type (replaces PartFeatureExtentEnum + the Set*Extent zoo)
type Extent interface{ isExtent() }
type DistanceExtent struct{ Dist param.Quantity }
type ToFaceExtent   struct{ Face Ref; Extend bool }
type ToNextExtent   struct{}
type ThroughAllExtent struct{}
type FromToExtent   struct{ From, To Ref }

type Extrude struct{ def ExtrudeDefinition; result feature.Result } // the realized feature
func (e *Extrude) recompute(ctx context.Context, in feature.Inputs) (feature.Result, error) {
    prof, ok := in.resolve(e.def.Profile);  if !ok { return sick("profile lost") }
    body, lineage, err := ops.Extrude(prof.(*sketch.Profile), e.def.Extent, e.def.Operation, e.def.Taper, in.bodies)
    // lineage tags start/end/side faces → reference keys for downstream picks
    return feature.Result{Bodies: body, Lineage: lineage}, err
}
```

- **Extents are a sealed Go sum type**, not an enum + parallel `Set*Extent` methods
  + nullable fields. The compiler enforces that exactly one extent kind is present —
  the COM `PartFeatureExtentEnum` + `ExtentTwo`/`ExtentTwoType` footguns vanish.
- **Result faces get lineage** (start/end/side) → reference-keyed, so a later fillet
  can pick "the end face of this extrude" durably (modeling/01, ADR-0010).
- **Registration**: `func init(){ registry.Features.Register(extrudeKind) }` — adding
  extrude is a package + one blank import (core/07); its ribbon button and edit panel
  (reflecting `ExtrudeDefinition`'s tags) appear for free (core/09).

## The rest of the sketched features

Same triangle; each `recompute` calls the matching kernel op:

| Feature | Definition highlights | Kernel op | Phase |
|---|---|---|---|
| **Revolve** | `Axis Ref`, `Angle Quantity`, full/partial, `Operation` | `ops.Revolve` | A |
| **Hole** | placement (linear/concentric/sketch), type (drill/cbore/csink), `Tap` info | `ops.Hole` | A |
| **Rib** | open `Profile Ref`, `Thickness`, to-next | `ops.Rib` | A/B |
| **Sweep** | `Profile`, `Path Ref`, orientation, twist, guide rail | `ops.Sweep` | B (NURBS) |
| **Loft** | `Sections []Ref`, `Rails []Ref`, conditions | `ops.Loft` | B (NURBS) |
| **Coil** | `Profile`, axis, pitch/height/revolutions/taper | `ops.Coil` | B |

Each ships `XDefinition` + `XFeatures.Add`/`AddByXxx` convenience + `XFeature` +
recompute — uniform, so learning extrude teaches all of them (parametric-cad §3).
Sweep/loft/coil wait on kernel **Phase B** (NURBS); the rest run on **Phase A**.

## Derived & imported features

- **Derived part/component**: an associative copy pulling geometry from a source
  document (scale/mirror/body selection), updating when the source changes — a
  `Feature` whose input is another document's content (reference-tracked, core/05).
- **Non-parametric base / imported**: a body from translation (M17) wrapped as a
  base `Feature` that downstream features can edit. On the Go-native kernel these
  arrive via the translator framework (iteration 4) and are healed (kernel Phase D).

## What got better than COM

- **Sealed extent sum types** replace enum + nullable-field + `Set*Extent` sprawl —
  illegal states unrepresentable.
- **One `Definition` struct** is create + edit + serialize + inspect + RPC.
- **Self-registration** replaces the `ExtrudeFeatures`/`CommandManager` wiring.
- **`Ref` inputs + lineage** make downstream picks robust across rebuilds by
  construction (ADR-0010), instead of the COM `GetReferenceKey` dance per call site.
