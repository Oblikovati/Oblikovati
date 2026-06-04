# ADR-0025 — Anisotropic material properties (isotropy class + orthotropic elastic group)

**Status:** accepted (user decision, 2026-06-04) · **Relates to:**
[ADR-0022](ADR-0022-materials-appearances.md) (materials subsystem),
[ADR-0024](ADR-0024-embedded-material-catalog.md) (the catalog this data ships in),
[ADR-0018](ADR-0018-apache-api-contract-module.md) (API split this followed).

## Context

The `Material` model (ADR-0022) stores **scalar, isotropic** structural properties:
`Mechanical{YoungsModulus, PoissonsRatio, YieldStrength, UltimateTensileStrength}`. For
an eventual FEA capability this is exactly right for metals, alloys and bulk plastics —
isotropic linear-static / modal / linear thermal-stress analysis needs only E, ν, ρ and α
(shear modulus is derived, `G = E/2(1+ν)`).

It is **wrong** for materials whose stiffness is direction-dependent: woods (orthotropic;
transverse modulus ≈ 1/20 of longitudinal) and fibre-reinforced composites (a UD carbon
lamina is ~135 GPa axial vs ~10 GPa transverse). Feeding a single-axis E to a solver as
isotropic produces an incorrect stress field, not a conservative one. We want the catalog
**simulation-ready now**, before solver work starts, so the data isn't re-sourced later.

## Decision

1. **Add an isotropy class + an optional orthotropic elastic group, contract-first.** New
   in `api/types` (Apache-2.0): `IsotropyClass` (`isotropic` / `orthotropic` /
   `transversely-isotropic`; empty == isotropic, with an `Anisotropic()` predicate) and
   `AnisotropicElastic` — the nine independent orthotropic constants (E1,E2,E3, G12,G23,G13,
   ν12,ν23,ν13) plus directional CTE (α1,α2,α3) for orthotropic thermal stress, in the
   material principal axes. `contract.Material` gains `IsotropyClass()` / `Anisotropic()`;
   `wire.MaterialInfo` gains the two fields (so a future FEA add-in reads them over the
   wire). Then the GPL impl: `MaterialSpec` / `MaterialRecipe` fields, accessors, the
   YAML `isotropy:` / `orthotropic:` keys (orthotropic is a pointer so isotropic materials
   serialize no block).

2. **Scope = elastic + thermal only.** The group carries stiffness and CTE — enough for
   orthotropic linear-static and thermal-stress FEA. **Out of scope** (separate future
   gaps, unchanged by this ADR): plastic hardening curves, and directional/failure
   strength allowables for composites (Tsai-Wu/Hashin) — those are criterion- and
   layup-specific and will be added when a solver and failure model are chosen. Until
   then `Mechanical.YieldStrength`/`UTS` remain the equivalent scalar strengths.

3. **Only genuinely anisotropic catalog entries are tagged.** Woods → orthotropic
   (constants derived per species from its longitudinal modulus via the FPL Wood Handbook
   average elastic ratios). UD carbon/glass/aramid laminae → transversely-isotropic;
   balanced-weave carbon, G10/FR4, plywood and OSB → orthotropic. **Random-mat composites
   stay isotropic** (chopped-strand fiberglass, SMC, carbon-filled nylon, MDF are in-plane
   isotropic), as do all metals and bulk plastics. Sources are cited in each catalog file.

## Consequences

- The catalog is FEA-ready for the realistic first target (isotropic linear analysis) **and**
  for orthotropic linear/thermal analysis of wood and composites, with no later re-sourcing.
- Zero migration: empty class = isotropic, so every existing material and serialized file is
  valid unchanged; the orthotropic block is omitted for isotropic assets.
- Test-guarded: an anisotropic material must carry a complete positive elastic group, and a
  populated group must declare a non-isotropic class (a forgotten tag fails the build).
- The orthotropic constants for woods are **derived from standard FPL ratios**, not per-
  specimen measurements; transverse CTE values are representative for timber. Documented in
  the catalog headers; refine against species-specific data if a wood-specific analysis
  needs it.
