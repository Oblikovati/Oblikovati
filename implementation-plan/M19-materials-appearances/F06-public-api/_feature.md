# M19 · F06 — Public API: appearances / materials / assign / physical-properties

> **Backfilled 2026-06-04 from shipped code.** See REPORT.md D-03.

## Scope (in)

`/api` surface for add-ins: list/get/create/update appearances & materials, assign to
model, read physical properties.

## Code (as built)

- `api/contract/{appearance.go,material.go}` (typed interfaces).
- `api/wire`: `appearances.{list,get,create,update}`, `materials.{list,get,create,update}`,
  `model.{assignMaterial,assignAppearance,physicalProperties}`.
- `addin/router` handlers + `api/client` typed group.

## PBIs

| PBI | Title | Grade |
|-----|-------|-------|
| [PBI-197](PBI-197-public-api.md) | Materials/appearances wire + client + router | M✅ G n/a U n/a |
