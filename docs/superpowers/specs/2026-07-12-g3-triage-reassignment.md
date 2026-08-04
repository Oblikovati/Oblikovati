# G3 "generic invalid-solid" triage — reassignment (2026-07-12)

**What this is:** the diagnosis pass the greening roadmap calls for before committing effort to
G4–G6 — run every currently-faulty corpus case, capture *why* its solid is invalid (the fillet
feature's `Health().Reason`), and reassign each to a real engine family or flag it as a genuine
build bug. Not a fix package; its output is scope + the next target.

**Method:** a throwaway sweep (`TestG3TriageDump`, since deleted) over all 475 corpus cases via the
same path as `ScoreCase`/`RunCase`, dumping `grid/case | mode | bodies/vol | reason` for every
`FailFaulty` case. Reason-string frequency is the classifier. Branch `feat/occt-blend-parity-corpus`
@ `05dd3236`; scoreboard TOTAL PASS=31, **217 FailFaulty**.

## Reassignment of the 217 FailFaulty

| family | count | reason signature | where it goes |
|---|--:|---|---|
| **G4** miter / corner-face-planar | **93** | "corner face must be planar" (31), "miter corner's shared face must be planar" (29), "…outer face must be planar" (22), "miter edge has no outer face opposite the shared face" (10), "must share a face to miter" (1) | miter reconstruction for non-planar shared/outer/corner faces + the no-outer-face degenerate |
| **G5** n-way / mixed-radius / arc-end | **35** | "…is not a supported blend (need 3 edges at a trihedral vertex…)" (11 across 2/4/5/6-face vertices), "arc end is not a cylinder/side tangent vertex" (9), "edge endpoint has no end face to round" (4), "…must use one radius there (got 1 and 0.5)" (2) | multi-edge corner blends (distinct from the single-edge runout just closed) |
| **G6** curved-neighbour | **34** | "…borders a curved (cylinder/cone/sphere/torus/bspline) face is not yet supported" | filleting an edge adjacent to a curved face |
| **G7** edge bounds ≠ 2 planar | **5** | "edge bounds 1 faces, need 2" | topology one-offs |
| **BUG** inconsistent orientation | **22** | "result is not a valid solid [inconsistent orientation at edge N …]" — **builds a body (vol>0) but with wrongly-wound faces** | **real assembly build bug** (see below) |
| **BUG** invalid, empty reason | **4** | "result is not a valid solid []" — builds, invalid, no specific reason | real build bug (validity fails with no named cause) |

### G4 (93) — the largest single lever
`bfuseblend/{A2–A9,B1–B7}`, `complex/{C5,C6,E5,F4,F7,G1,G2}`, `encoderegularity/{A1,A4}`,
`simple/{B3,B7,C2,C6,C8,D1,D5,D9,E4,E6,E8,F1,F3,G4,G8,H1,H7,H9,I2,I4,I8,J1,J2,J3,J4,J5,J6,J8,J9,K2,K3,K4,L8,L9,M3,M5,M6,M8,M9,N1,N2,N4,N7,N8,O1,O2,O4,O5,O7,O9,P2,P3,P5,P7,R7,S2,S5,S8,T2,T5,T8,U2,W3,W4,W5}`,
`tolblend_simple/{B7,C2,C3,D4}`.

### G5 (35) — multi-edge corner blends
`bfuseblend/A1`, `complex/{B9,C2,D2,E3}`,
`simple/{A4,F5,F7,G3,G6,H3,H4,H5,I9,K1,P9,R8,U6,V9,W6,W8,W9,X5,X8,Z1}`,
`tolblend_simple/{A1,A2,A9,B1,B2,B3,D3,E6,E7,E8}`.

### G6 (34) — curved-neighbour
`complex/{C1,E8,F1}`,
`simple/{B6,C1,C5,C9,D4,D8,E3,E5,E7,E9,H2,I1,I5,I6,I7,M4,M7,N3,N9,O3,O6,O8,P1,P4,P6,Q2,U5,V6}`,
`tolblend_simple/{B4,B8,C7}`.

### G7 (5)
`simple/{F4,G5,G7,G9}`, `tolblend_simple/C8`.

### Real build bugs (26) — the recommended next fix
**Inconsistent orientation (22, all `simple/`):** `K6,K9,L1,L3,L4,L6,L7,R9,S1,S3,S4,S6,S7,S9,T1,T3,T4,T7,T9,U4,X3,Y1`.
Each *builds* a single body with positive volume, then the validity gate rejects it for
inconsistently-oriented faces ("inconsistent orientation at edge N"). This is a face-winding defect
in the fillet assembly, **not** a missing feature. The cases cluster by shape/scale (K/L ≈ 1.0M u³
box-like; S/T ≈ 1e4–1.6e5), strongly suggesting **one (or few) root cause(s)** in the fillet
face-orientation logic — a classic "one fix unlocks many."
**Empty reason (4):** `simple/{F6,T6,U3}`, `tolblend_simple/D5` — build, invalid, no named cause;
diagnose alongside (may share the orientation root or be a distinct closure gap).

## Greened since the roadmap's original G3 list was written
`A6, A8, K7, W1` (G1 corner-solve orientation fix), `V1` (runout setback, this session), `X9`
(runout first slice). The remaining originally-G3-listed cases are absorbed into the buckets above.

## Recommendation
1. **Next fix = the 22 "inconsistent orientation" build bugs** — highest value/lowest risk: the
   fillets already build, so this is a focused face-winding root-cause hunt (systematic-debugging),
   likely fixing many at once, and it un-blocks correct area comparison for those shapes. Do the 4
   empty-reason cases in the same pass.
2. **Then G4 miter blends (93)** — the largest feature gap and the core ADR-0050 Phase-6 corner
   machinery; route the non-planar corner geometry through `geometry-math-advisor`.
3. G6 (34) and G5-multi-edge (35) follow; G7 (5) mops up.
