# ADR-0040 — External geometric references for externally-authored topology selections

**Status:** Accepted — design (2026-06-21); the resolver landed as a spike (`kernel/topo`),
the feature/API/exporter wiring is the M8 follow-up (see Scope). · **Builds on / refines:**
[ADR-0018](ADR-0018-api-implementation-split.md) (the API/implementation split this must
follow), the M31 topological-naming work (F06 tiered binding `model/identity/binding.go`,
F07 versioned encoding). **Touches (when implemented):** `kernel/topo` (the resolver,
prototyped in `kernel/topo/geometric_ref.go`), the dress-up resolution in
`model/feature/{fillet,chamfer,dressup,hole_boss,face_fillet}.go` and `resolveEdges`,
the recipe codecs in `model/feature/serialize_dressup.go`, and new value types in
`../Oblikovati.API/types`.

## Context

Dress-up and placed features — fillet, chamfer, shell, draft, thread, and face-placed
holes/bosses — select **realized** edges/faces of the running body. Each selection is
persisted as an Oblikovati **lineage reference key**: a kind byte plus a `Feature:Role#Index`
lineage string (`kernel/topo/lineage.go`), and re-resolved each recompute by exact match
(`Body.FindFaceByKey` / `FindEdgeByKey`, via `resolveEdges`).

The **NX exporter (M8)** must reproduce these selections, but it **cannot synthesize a
lineage key**: the lineage strings are produced by Oblikovati's *own* feature recompute
and have no counterpart in NX's data. An external author knows only *geometry* — where the
selected face/edge sits.

M31 already degrades binding gracefully (F06): on an exact miss it recovers a surviving
sibling ancestrally, or — when several share the parent — by geometric nearness to the
key's mint-time **anchor**. But that path is unavailable to an external author for two
reasons:

1. The anchor and parent hints are **deliberately not serialized**. F07 versioned the key
   encoding yet keeps the *drift-prone* anchor out of the identity bytes (`refkey.go`): a
   key reloaded from disk degrades to exact-only. The hints are an in-memory session
   optimisation.
2. The geometric tier still requires a **parent lineage** to narrow candidates
   (`fallbackMatch` returns `MatchNone` when `!k.parent.ok`). The exporter has no lineage
   at all.

So neither the identity key nor the M31 fallback can carry an externally-authored
selection across a recompute. We need a path that binds from geometry *alone*.

## Decision

Introduce a **serialized, parallel "external geometric reference"** — distinct from the
identity-bearing `RefKey` — plus a kernel resolver that binds it by geometry.

1. **Value types (API, `api/types`)** — a selection descriptor in model space:
   - `GeometricFaceRef{ Centroid, Normal }` — a representative point and outward unit normal.
   - `GeometricEdgeRef{ Midpoint, Direction }` — midpoint and sign-agnostic direction.
   (Richer fields — face area, adjacent-face normals, edge length — are a disambiguation
   follow-up; see Consequences.)

2. **Kernel resolver (`kernel/topo`)** — prototyped in this spike as
   `Body.FindFaceByGeometry` / `FindEdgeByGeometry`: take the candidate of the right kind
   whose centroid/midpoint is within tolerance **and** whose normal/direction aligns, and
   return it only when it is **unambiguous**. An equally-near aligned tie returns "lost"
   rather than binding the wrong entity — the same "defensible or lost" rule as the M31
   geometric tier. `DescribeFace`/`DescribeEdge` produce a descriptor from an entity (the
   inverse), for converting an Oblikovati-side selection and for tests.

3. **Recipe encoding (`serialize_dressup.go`)** — each selection slot persists **either**
   a lineage key (Oblikovati-authored, unchanged) **or** a geometric ref (externally
   authored): a small tagged union per edge/face. Existing files are untouched.

4. **Resolution routing** — when a slot is a geometric ref (or a lineage key that misses),
   resolve via the geometric resolver; map a geometric hit to **`health.Warning`**
   (auto-healed, flagged for the user to confirm), exactly like the M31 fallback. A pure
   lineage key still resolves exactly first, then through the existing M31 tiers.

This stays **non-identity**: a geometric ref never feeds a key's identity bytes, preserving
F07's rule that the drift-prone anchor must not destabilise identity. It only recovers a
selection at recompute.

### Why not extend `RefKey`/the M31 anchor

Serializing the anchor into the key would re-introduce exactly what F07 rejected: a
drift-prone value inside the identity. And the M31 geometric tier is *lineage-narrowed* by
design (it disambiguates same-parent siblings); an external author has no lineage to narrow
with. A separate, explicitly-geometric ref keeps both systems honest.

## Spike evidence

`kernel/topo/geometric_ref.go` + `kernel/ops/geometric_ref_spike_test.go` (this branch)
demonstrate the core claim: a descriptor captured on one box **re-resolves every face and
every edge on an independently built, geometrically identical box** (different object graph,
no shared lineage), and a far-away descriptor **misses honestly** (no wrong bind). This
de-risks the central question — geometric re-binding from a serialized descriptor is
feasible and safe for the common cases.

## Consequences

- **Ambiguity on symmetric / repeated geometry** (e.g. a pattern of identical holes, a
  symmetric flange) yields several equally-near aligned candidates; unique-or-fail then
  *loses* them rather than guessing. Mitigations, in order of preference: (a) a richer
  descriptor (face area + sorted adjacent-face normals; edge length + endpoint vertices);
  (b) the exporter selecting the source feature *before* its pattern so only the seed is
  geometrically bound; (c) accepting loss with a clear warning. The seed-before-pattern
  ordering already falls out of M5 (patterns reference a source feature, not faces).
- **Tolerance** is an absolute model-space distance; the centroid-of-vertices is exact for
  planar faces and approximate for curved ones (the normal filter covers the difference). A
  follow-up can use the true area-weighted centroid.
- **Performance** is O(faces/edges) per selection — negligible (dress-ups select a handful;
  an index can be added if needed).
- **Additive and safe**: Oblikovati-authored lineage keys are unchanged; they still bind
  exactly first and fall back through the M31 tiers. Only externally-authored selections
  take the new path.

## Scope: spike vs full M8

This ADR + the `kernel/topo` PoC are the **spike**. Full M8 then needs, in ADR-0018 order:
the `api/types` value types + `api/wire` recipe DTOs; the `serialize_dressup.go` tagged-union
encoding; routing `resolveEdges`/`FindFaceByKey` consumers through the geometric resolver on
a geometric ref; the exporter's Tier B emission (NX edge/face → descriptor); and oracle
tests (export a known filleted/holed NX part → reopen → assert the dress-up bound and the
volume matches).
