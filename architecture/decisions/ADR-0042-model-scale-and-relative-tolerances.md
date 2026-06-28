# ADR-0042 — Model scale & relative tolerances: a kernel resolution centred on the working scale

**Status:** Accepted — shipped in M35 / M37 (model-relative tolerances threaded through the kernel, PR#1427) · **Builds on / refines:**
[ADR-0020](ADR-0020-yaml-git-friendly-document-format.md) (the `.obk` document that persists a
document's units) and the Units-of-Measure work (Oblikovati/Oblikovati#146) — this ADR governs
*how the kernel's numerical tolerances and working scale relate to the document unit*, so the
modeller stays faithful at extreme scales (semiconductor, urban). **Touches (planned):** the
kernel tolerance call-sites (`kernel/ops/*.go`, `kernel/geom/*.go`), a new resolution/tolerance
context, and — in the later phase — coordinate storage, persistence (`model/**/serialize*.go`),
assembly placement (`model/compdef`) and exchange (`kernel/exchange/**`).

## Context

Every length in the kernel is stored in **centimetres as `float64`** (`model/param`: cm is the
factor-1 database unit). The document unit chosen in Document Settings is a **display-time**
concern only (#146, #1239, #1241): it changes the number shown, never what is stored.

The kernel's geometric tolerances are a **mix**, and most are **absolute constants anchored at
cm**:

| Constant | File | Value | = |
|---|---|---|---|
| boolean weld | `kernel/ops/boolean.go` | `1e-6` cm | 10 nm |
| CSG epsilon | `kernel/ops/csg.go` | `1e-7` cm | 1 nm |
| CSG weld grid / on-line | `kernel/ops/csg_body.go` | `1e-6` cm | 10 nm |
| surface sew default | `kernel/ops/sew.go` | `1e-4` cm | 1 µm |
| hole/edge weld | `kernel/ops/union_holes.go`, `edge_discretize.go` | `1e-9` cm | 0.01 nm |
| convex-hull tol | `kernel/ops/convexhull.go` | `1e-9 × boundsDiagonal` | *relative* |
| surface query | `kernel/geom/evaluator_surface_query.go` | `1e-9 × max(1, best)` | *relative* |

Two problems follow.

**1. Absolute, cm-anchored tolerances are scale-blind.** A semiconductor design authored at
nm/pm scale has its real geometry **welded out of existence** — every coordinate is closer than
the `1e-6` cm boolean/CSG tolerance, so booleans merge distinct vertices, and `sew` (1 µm) is
catastrophic. The cm anchor is arbitrary: it was tuned empirically for ~1–100 cm parts, where
`1e-6` cm is ~`1e-8` relative. At the other extreme (urban/road, tens of km) the same constants
are needlessly tight and inconsistent with the *relative* tolerances a few ops already use.

**2. The system is internally inconsistent.** `convexhull` and `evaluator_surface_query` already
scale their tolerance by the model bounds; the boolean/CSG/sew paths do not. There is **no
single source of truth** for "how close is coincident in this model".

### A correction to the obvious framing

It is tempting to say "make the document unit the kernel's `1.0` and the dynamic range follows."
`float64` is *floating*: its ~15.95 significant-digit **relative** precision is independent of the
exponent, so `1 pm` stored as `1e-10` cm carries the **same** ~16 digits as `1.0` in a pm base
unit. Rebasing the unit does **not**, by itself, widen the dynamic range of a single stored
coordinate.

What a working-scale base unit *does* buy is real but different:

- **Conditioning.** When coordinates are `O(1)` rather than `O(1e-10)`, subtraction of nearby
  values and comparison against epsilons stay well-behaved; you stop testing `1e-10` coordinates
  against an absolute `1e-6` tolerance.
- **Scale-appropriate constants for free** — but only if those constants are themselves expressed
  relative to the working scale.

So the defect that actually breaks the extremes is the **absolute, cm-anchored tolerances**, not
the storage type. And the hard ceiling is `float64` itself: ~16 significant digits ≈ **15 orders
of magnitude** of usable span *within one model*. You cannot have a pm feature on a km part in a
single body; you can, and should, support a pm **document** and a km **document** each at full
fidelity.

## Decision

Adopt the model that mature kernels (Parasolid's size-box + linear precision, ACIS's
`SPAresabs`) converge on, in two phases.

### Phase 1 — A single model resolution; relative tolerances (no persistence change)

1. Introduce one **`Resolution`** concept: `resolution = εRel × modelSize`, where `modelSize` is
   the operand's bounding-box diagonal (with a sane floor so a degenerate/empty operand still has
   a positive resolution), and `εRel` is a small dimensionless constant (the current `1e-9`
   bounds-relative ops set the precedent).
2. Thread it through the kernel ops as a small **tolerance context** rather than reading global
   literals: every weld/coincidence/on-line test derives its tolerance from the resolution of the
   geometry it is operating on.
3. **Replace the absolute constants** in `boolean.go`, `csg.go`, `csg_body.go`, `sew.go`,
   `union_holes.go`, `edge_discretize.go` with resolution-derived tolerances; fold the already
   relative `convexhull`/`surface_query` ops onto the same helper.
4. Gate with **scale-sweep regression tests**: the same part modelled across a range of unit
   scales must produce the same topology and volume (up to relative tolerance). This is the
   acceptance criterion and the thing the current design fails.

Phase 1 touches only the kernel — no document, persistence, assembly or exchange change.

**What Phase 1 actually delivers (implementation finding, #1242–#1244).** The model-relative
tolerances make a representative filleted boolean part scale-invariant from **1 µm to 1 km**
(the `TestScaleSweepInvariance` gate). It fully resolves the **urban/large** case and reaches
down to micron features. It does **not**, on its own, reach the **nm/pm semiconductor** extreme:
below ~1 µm the limiting factor is no longer a kernel weld tolerance but the **fundamental
vector-normalisation epsilon used in primitive construction** — building a plane/box at pm scale
takes a cross product of pm-length edge vectors (~1e-20 magnitude) which underflows the absolute
`math.UnitVector3FromVector` floor (1e-9), so the primitive is born degenerate before any
tolerance applies. That epsilon is a pure vector operation with no model-size context to scale
by; the only correct fix is to keep coordinates `O(1)` — i.e. **Phase 2**. Attempting to bridge
the gap by normalising operands at an op boundary fails for the same reason (the similarity
transform itself underflows on native pm geometry). So the semiconductor case is **re-scoped to
Phase 2**, not Phase 1.

### Phase 2 — Working-scale storage centred on the document unit

1. Make the kernel's working scale track the **document unit** so coordinates are `O(1)–O(1e3)`
   (the conditioning win above). Storage, persistence, assembly placement and exchange all carry
   or convert the unit:
   - **Persistence:** the `.obk` already records the document unit (ADR-0020); values are stored
     in (or normalised to) the working scale and the unit travels with them.
   - **Assembly mixing:** a component placed into an assembly of a different unit converts at the
     placement boundary (the transform already exists; this adds a scale term).
   - **Exchange:** STEP is mm-based; the exchange layer remains the single place lengths leave the
     working scale (it already owns unit conversion, #146).
2. Document the **single-model span ceiling** (~15 orders) in the UI/docs so a user mixing pm
   features onto a km part gets a clear diagnostic rather than silent merging.

Because working-scale storage keeps coordinates `O(1)` regardless of the document unit, Phase 2
is also what unlocks the **nm/pm semiconductor** scales that Phase 1 cannot reach (see the Phase 1
finding above): a pm document's coordinates become `O(1)` working values, so primitive
construction and the vector-normalisation epsilon never see 1e-10 magnitudes.

Phase 2 is the larger, multi-PR, multi-repo change and is sequenced after Phase 1 proves the
relative-tolerance core.

## Consequences

- **Positive:** correct behaviour across ≥15 orders of magnitude of *document* scale; one source
  of truth for coincidence; removal of ~8 magic constants; an explicit, tested scale-invariance
  guarantee.
- **Negative / cost:** every kernel op that compares distances must take a resolution; Phase 2
  ripples into persistence/assembly/exchange and needs migration care for existing `.obk` files.
- **Bounded, not infinite:** `float64` still caps a *single model* at ~15 orders; this ADR makes
  the *document* scale free, not the per-model span. A future need for pm-on-km in one model would
  require a different number type (e.g. fixed-point integers) and is explicitly out of scope.

## Alternatives considered

- **Fixed-point / integer nanometre coordinates** (the EDA approach): exact at one scale, but
  loses the curved-surface NURBS math the kernel is built on and would not serve mechanical CAD.
  Rejected.
- **Leave tolerances absolute, just add more units** (the status quo after #1241): keeps the
  display-only fix but does nothing for the actual welding defect. Rejected — this ADR exists
  because that is insufficient.
- **Rebasing storage only, keep absolute constants:** improves conditioning but the constants are
  still wrong relative to the model; the scale-sweep test would still fail. Rejected as a
  half-measure; relative tolerances (Phase 1) are the load-bearing change.

## Rollout

Tracked by milestone **M35: Model Scale & Relative Tolerances**.

**Phase 1 — relative tolerances (kernel-only) — DONE:**
- #1242 — kernel resolution & tolerance context (`resolution = εRel × modelSize`) ✅
- #1243 — migrate boolean/CSG/sew/weld ops off the absolute cm constants ✅
- #1244 — scale-sweep regression tests (µm→km topology/volume invariance) — the acceptance gate ✅

**Phase 2 — working-scale storage (centred on the document unit) — DONE:**
- #1245 — working-scale storage (`param.UnitsOfMeasure.workingScale`; `Quantity.Value` in working units) ✅
- #1246 — persist values in working scale + unit round-trip (`.obk`, with automatic migration) ✅
- #1247 — assembly mixing: convert at the placement boundary (`childWS/ownerWS` scale term) ✅
- #1248 — exchange: working-scale ↔ STEP mm (`TargetUnitMM` = working-unit mm) ✅
- #1249 — span-ceiling diagnostic (`geom.SpanCeilingWarning` / `PartComponentDefinition.FeatureScaleWarning`) ✅
- #1259 — activation: auto-centre + working-unit report contract ✅

### Activation & the report contract (#1259)

Turning the storage on for users settled on a **working-unit report contract**: the kernel stores
and reports every geometric quantity in the document's working length unit (the centimetre by
default, so existing documents are byte-identical), and unit conversion happens only at the two
boundaries that already own it — **input** (the parser/expression solver converts an explicit-unit
value like `5 mm`; a bare number is taken in the working unit) and **output** (the user picks the
file unit on export). The working scale is exposed (`wire.DocumentUnitsInfo.WorkingScaleCm`) so an
add-in can recover centimetres from a raw geometry quantity by the appropriate power. This avoids a
fragile per-call-site "convert everything to cm" audit and keeps every reported value internally
consistent.

A fresh document **auto-centres** its working scale on its chosen length unit
(`PartComponentDefinition.SetLengthUnit` / `SetUnits` → `param.RecommendedWorkingScale`): band
units (mm…ft) keep the centimetre scale; extreme units (µm/nm/pm, km) centre on themselves so
coordinates stay O(1). Once geometry exists the scale is frozen (changing it would reinterpret
stored coordinates), so a later unit change is display-only. This is what makes the nm/pm
semiconductor (and km urban) scales modellable end to end.

### Span-ceiling diagnostic (#1249)

float64 holds ~15.95 significant decimal digits, so **one model** spans at most ~15 orders of
magnitude before a small feature falls below the representable floor and is silently merged. A
document is free to sit anywhere on the scale (a pm part and a km part are each fine on their own),
but a feature more than ~10¹⁵ smaller than the model extent cannot coexist with it. The kernel
surfaces this rather than merging silently: `geom.FeatureResolvable` / `geom.SpanCeilingWarning`
compare a candidate feature size against the model's coincidence resolution
(`Resolution.Weld() = εRel × extent`), and `PartComponentDefinition.FeatureScaleWarning` wraps it
for the head/API to show. The remedy the message gives is to model at a working unit nearer the
feature scale (Phase 2 makes that free) or split the design across documents.
