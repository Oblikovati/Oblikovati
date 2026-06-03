---
milestone: M20
name: Feature Completion & Geometry Parity
status: in-progress
---

# M20 — Feature Completion & Geometry Parity

The Extrude feature (M08·PBI-092) established the canonical feature pattern: the
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`
surface, `addin/router` handler, `.obk` serialize round-trip, UI/ribbon hook, and
executable tests. This milestone drives **every remaining Inventor `*Feature` type
to the same standard, with real generated geometry** — not just deferred recipes.

Most part/surface features already exist as triangles whose geometry is *deferred*
because the kernel's core solid operations were postponed (general intersecting
booleans, NURBS sweeps, rolling-ball fillets — "Phase B/C/D, NotYetImplemented").
This milestone implements those kernel operations first, then completes the
features that sit on them, then adds the feature families with no presence yet:
**sheet metal**, **plastic parts**, and the **misc model features** (move,
direct-edit, reference, finish, mark, simplify, iFeature, presentation mesh).

## Goals

- Land the deferred kernel ops (intersecting booleans, swept surfaces, fillet,
  local face ops, body transform) so existing deferred features emit real solids.
- Complete the full sheet-metal environment, feature set, and flat-pattern unfold.
- Complete the plastic-part feature set.
- Complete the remaining misc model features for API/persistence parity.

## In scope (every remaining `*Feature` in the Inventor API)

- **Kernel enablers:** intersecting booleans, swept/lofted/revolved/coil surfaces,
  rolling-ball fillet/chamfer, local face ops, rigid body transform.
- **Geometry-completion** of deferred M08/M09/M10 features (Revolve, Sweep, Loft,
  Coil, Rib, Fillet, Chamfer, Shell, Draft, Hole, Boss, Combine, Split, face edits,
  Thicken, patterns, mirror).
- **Sheet metal:** environment+rules; Face/Flange/ContourFlange/ContourRoll/Hem/
  Bend/Fold/LoftedFlange; Corner/CornerChamfer/CornerRound/CornerSeam; Cut/Rip/
  PunchTool/CosmeticBend/Unfold/Refold; FlatPattern + DXF.
- **Plastic parts:** Boss(plastic)/Emboss/Grill/Lip/Rest/RuleFillet/SnapFit; Decal.
- **Misc:** Move(body)/DirectEdit/Reference/Client/Finish/Mark/Simplify/Unwrap/
  ModelTolerance/iFeature/PresentationMesh.

## Out of scope (handled elsewhere)

- Assembly features (M11/M12), drawing of features (M14), full materials (M19).

## Exit criteria

- Every Inventor `*Feature` type has a `Definition → Add → Feature(+Proxy)` triangle,
  a `/api` surface, a `.obk` round-trip, and tests green in `make ci`.
- The deferred features that depend on landed kernel ops now generate validated
  manifold geometry (no longer Warning-deferred) and recompute on parameter change.
- A sheet-metal part unfolds to a correct flat pattern and exports to DXF.

## Depends on

M08, M09, M10 (and the M07 kernel/ops foundation they extend).

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Intersecting Booleans](F01-intersecting-booleans/_feature.md) | 2 | Face-splitting solid/solid boolean (Phase-C); unblocks Cut/Split/Hole/Combine/Emboss/sheet-metal/plastics. |
| **F02** | [Swept-Surface Generation](F02-swept-surface-generation/_feature.md) | 2 | Surfaces of revolution + sweep/loft/coil → real solids (Revolve/Sweep/Loft/Coil/Rib). |
| **F03** | [Fillet & Chamfer Geometry](F03-fillet-chamfer-geometry/_feature.md) | 2 | Rolling-ball fillet, chamfer faces, variable/setback; corner-round. |
| **F04** | [Local Face Operations](F04-local-face-ops/_feature.md) | 2 | Move/offset/delete/replace face, draft, shell, thicken real geometry. |
| **F05** | [Sheet-Metal Environment & Rules](F05-sheet-metal-environment/_feature.md) | 1 | `SheetMetalComponentDefinition`, thickness/radius/K-factor styles, unfold methods. |
| **F06** | [Sheet-Metal Wall & Bend Features](F06-sheet-metal-wall-bend/_feature.md) | 3 | Face, Flange, ContourFlange, ContourRoll, Hem, Bend, Fold, LoftedFlange. |
| **F07** | [Sheet-Metal Corner Features](F07-sheet-metal-corners/_feature.md) | 1 | Corner, CornerChamfer, CornerRound, CornerSeam. |
| **F08** | [Sheet-Metal Modify & Cosmetic](F08-sheet-metal-modify/_feature.md) | 1 | Cut, Rip, PunchTool, CosmeticBend, Unfold, Refold. |
| **F09** | [Flat Pattern](F09-flat-pattern/_feature.md) | 2 | Unfold solver, bend allowance, flat extents, DXF/DWG export. |
| **F10** | [Plastic Part Features](F10-plastic-features/_feature.md) | 2 | Boss(plastic), Emboss, Grill, Lip, Rest, RuleFillet, SnapFit. |
| **F11** | [Cosmetic & Reference Features](F11-cosmetic-reference/_feature.md) | 1 | Decal, ReferenceFeature, ClientFeature, Mark, Finish. |
| **F12** | [Body Transform Features](F12-body-transform/_feature.md) | 2 | Rigid body transform op → Move(body), Copy, Mirror-body, real pattern/mirror duplication. |
| **F13** | [Direct-Edit & Simplify Features](F13-direct-edit-simplify/_feature.md) | 1 | DirectEdit, Simplify, Unwrap, ModelTolerance. |
| **F14** | [iFeature Catalog](F14-ifeature-catalog/_feature.md) | 1 | Extract/place catalog iFeatures. |
| **F15** | [Presentation Mesh Feature](F15-presentation-mesh/_feature.md) | 1 | `PresentationMeshFeature` + mesh→B-rep. |
