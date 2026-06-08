---
milestone: M24
feature: F04
name: Oracle gating & EDF regression
status: planned
---

# M24 · F04 — Oracle gating & EDF regression

Lock the milestone in with committed, automated guards so the freeform mesher can never silently
regress: fold / over-enclosure / per-face-area detectors as tests, a synthetic trimmed-NURBS OCC
fixture in the oracle suite, and the EDF model as an end-to-end volume + fold regression. These are
the metrics that exposed every failed attempt during diagnosis; making them permanent tests is the
exit gate.

## In scope

- A committed **fold detector** (adjacent triangles with opposing 3D normals) run on the oracle
  fixtures' faces — assert 0 on freeform faces.
- A committed **over-enclosure / per-face-area** check (mesh area vs densely-sampled reference;
  total volume vs OCC `getMass`).
- A synthetic **trimmed freeform-NURBS** OCC fixture (a B-spline patch with a hole) added to
  `testdata/occ/` via the generator.
- EDF.STEP wired as an end-to-end volume + fold regression (kept out of the public path per the
  clean-room rule; a local/CI fixture).

## Out of scope

- New geometry kinds; this is validation only.

## Key API contracts delivered

- Committed detectors + oracle fixtures (test infrastructure).

## Depends on

F02, F03, the OCC oracle (`kernel/exchange/step/occ_oracle_test.go`), the gmsh SDK ground truth.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-319](PBI-319-fold-area-detectors-fixture.md) | Fold + over-enclosure detectors and a trimmed-NURBS OCC fixture |
| [PBI-320](PBI-320-edf-end-to-end-regression.md) | EDF end-to-end volume + fold regression |
