---
milestone: M25
feature: F04
name: Orientation, validation & oracle gate
status: planned
---

# M25 · F04 — Orientation, validation & oracle gate

Finish the heal: orient each shell coherently outward, validate it (manifold/closed), and lock the
whole milestone with the OpenCASCADE volume oracle + an EDF end-to-end regression that runs the
healed geometry through the M24 mesher and checks it is fold-free and volume-correct.

## In scope

- Coherent shell orientation (every face outward) via shared-edge use directions + a global
  volume-sign flip; report non-orientable / non-manifold input.
- A validation pass (manifold, closed-where-expected) + a healing report (what changed).
- Oracle gate: healing a clean fixture is a no-op; healed EDF → M24 mesher → fold-free + volume OK.

## Out of scope

- The mesher (M24).

## Key API contracts delivered

- (internal) shell orientation + validation; healing report; oracle/regression tests.

## Depends on

F01–F03; M24 mesher; the OCC oracle.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-328](PBI-328-orientation-validation.md) | Coherent shell orientation + manifold/closed validation |
| [PBI-329](PBI-329-heal-oracle-edf.md) | Healing oracle + EDF end-to-end regression through M24 |
