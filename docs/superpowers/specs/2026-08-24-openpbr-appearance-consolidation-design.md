# OpenPBR Appearance Consolidation — M46

## Context

M45 ("OpenPBR Surface — Physically-Based Realistic Mode") landed `OpenPBRAppearance`
as an ADDITIVE material contract alongside the existing legacy metallic-roughness
`Appearance` (marked `Deprecated:` per ADR-0053, with removal explicitly scoped as "a
separate future MAJOR-version decision"). `Oblikovati.API` PR#282 (the `OpenPBRAppearance`
contract/wire/client) is already merged and released as `v0.151.0`. `Oblikovati` PR#2152
(the GPL implementation) is open, blocked on CI, not yet merged.

While fixing PR#2152's CI, the user decided to do that "future MAJOR-version decision"
now rather than defer it: **`OpenPBRAppearance` becomes the sole, primary material
system. The legacy `Appearance` is deleted, not kept as a facade.** PR#2152 is held —
this consolidation lands before anything further merges.

No shipped add-ins depend on the legacy wire API yet (confirmed with the user), so
breaking changes are acceptable. This is the deciding constraint that makes full
removal (rather than a compatibility-preserving derived view) the right call.

## Decisions

1. **Full removal, not a facade.** `material.Appearance`, `contract.Appearance`, the
   `appearance.*` wire methods/DTOs, and `head/ui/appearance_editor.go` are deleted
   outright — not kept as a thin derived view. One material system, not two, ever.
2. **Rename `OpenPBRAppearance` → `Appearance`** at the top level (types, wire DTOs,
   wire method names, client method group, MCP tool names) now that the plain name is
   free. The nine group types (`OpenPBRBase`, `OpenPBRSpecular`, `OpenPBRCoat`,
   `OpenPBRFuzz`, `OpenPBRThinFilm`, `OpenPBRTransmission`, `OpenPBRSubsurface`,
   `OpenPBREmission`, `OpenPBRGeometry`) **keep** their `OpenPBR` prefix — bare names
   like `Base`/`Coat`/`Fuzz` would be dangerously ambiguous in a `types` package that
   already has other generic names in play.
3. **`DefaultOpenPBRAppearanceID`** (currently `"openpbr-default"`) becomes
   `DefaultAppearanceID`, value renamed to `"default"` — reclaiming the legacy neutral
   appearance's old id now that it's free, rather than carrying the `openpbr-` prefix
   forward into persisted data indefinitely.
4. **Catalog YAML migrates to the native OpenPBR schema now**, not deferred. All 93
   `model/material/catalog/*.yaml` entries get rewritten from the old 5-scalar shape
   (albedo/metallic/roughness/emissive/opacity) to the full 9-group OpenPBR shape,
   generated via a one-time throwaway conversion script (reusing the about-to-be-deleted
   `migrateAppearance` mapping logic one last time), then hand-reviewed on a sample
   before the script is deleted.
5. **Old `.obk` documents and project libraries get a one-time load-time migration**,
   not dropped. Two distinct things both called "migration":
   - **Assignment fields** (which appearance id is assigned where): the YAML key names
     (`appearance`, `bodyAppearances`, `faceAppearances`) don't change — they're
     reclaiming names the `openPBRAppearance`-prefixed fields temporarily borrowed
     this session. An old file's assignment ids parse in with zero code changes and
     resolve correctly automatically for built-in catalog ids (same id, now natively
     OpenPBR-authored).
   - **Asset storage** (a document's embedded custom appearances, or a project
     library's saved custom ones) changed shape, so needs real conversion. Detected by
     shape-sniffing the raw YAML per embedded entry (old shape has a top-level
     `albedo`/`metallic` key; new shape has `base`/`specular`) and routing only
     old-shaped entries through a relocated `legacyAppearanceToSpec` converter (moved
     from `model/material` into `recipe.go`, since it's now migration-only logic, not
     a live code path). No new document-wide schema-version field needed.
6. **Sequencing**: GPL PR#2152 stays open and gets extended — F02/F04/F05/F06's commits
   land on the SAME `feat/m45-openpbr-host` branch, on top of M45's existing commits,
   before that branch merges (not a new GPL PR). The API side needs a genuinely NEW PR
   on `Oblikovati.API`, since PR#282 already merged and released as `v0.151.0` — F01's
   deletion+rename ships as a second PR/release on top of it (MINOR bump,
   `v0.151.0`→`v0.152.0`; pre-1.0 SemVer treats breaking changes as MINOR, MAJOR is
   reserved for 1.0+). Same two-repo shape as M45, just the GPL half is a continuation
   of an existing branch rather than a fresh one.

## Scope by repo

### `Oblikovati.API` (Apache-2.0)

- Delete: `contract.Appearance`, `wire.AppearanceInfo`/`CreateAppearanceArgs`/
  `UpdateAppearanceArgs`/`AssignAppearanceArgs`/`ListAppearancesResult`,
  `MethodModelAssignAppearance`/`MethodAppearances*`, `client.Appearances`.
- Rename (decision 2): `types.OpenPBRAppearance`→`Appearance`,
  `wire.OpenPBRAppearanceInfo`→`AppearanceInfo`, `CreateOpenPBRAppearanceArgs`→
  `CreateAppearanceArgs` (and Update/Assign/List siblings),
  `MethodOpenPBRAppearances*`→`MethodAppearances*`,
  `MethodModelAssignOpenPBRAppearance`→`MethodModelAssignAppearance`,
  `client.OpenPBRAppearances`→`client.Appearances`, MCP tool names
  `*_openpbr_appearance`→`*_appearance`.
- `contract.Material.AppearanceID()` unchanged (already just a string id).
- `DefaultOpenPBRAppearanceID`→`DefaultAppearanceID` (decision 3).

### `Oblikovati` (GPL)

- `model/material/assignment.go`: `AssignmentStore` collapses to a single chain —
  `partOpenPBRAppearance`/`bodyOpenPBRAppearance`/`faceOpenPBRAppearance` reclaim the
  plain names, legacy fields and `EffectiveAppearance`/`apprOrNil` deleted,
  `EffectiveOpenPBRAppearance`→`EffectiveAppearance` with its signature simplified
  back to always-returns-non-nil (`look.DefaultAppearance()` as the ultimate
  fallback) — the `ok bool` two-step existed only for the now-gone dual-chain
  fallback.
- `model/material/library.go` + `openpbr_library.go` + `assetset.go`: delete every
  legacy-`Appearance` method, rename every `OpenPBRAppearance`-suffixed method to
  drop the prefix. The `OpenPBRAppearance` struct replaces `Appearance` entirely.
- Delete `migrateAppearance` + its test (catalog loading no longer needs it — decision 4).
- Catalog: rewrite 93 YAML files (decision 4).
- `recipe.go`: field renames + the one-time legacy-shape migration (decision 5),
  tested against real old-format fixtures checked into `test-utilities/`.
- `addin/router/material.go` deleted; `addin/router/openpbr_material.go` renamed/merged
  in, keyed off the renamed wire methods.
- `head/ui/appearance_editor.go` deleted; `head/ui/openpbr_appearance_editor.go`
  becomes the Materials window's only appearance tab (drop the tab switcher).
- Every existing test referencing the old names/APIs updated — real mechanical churn
  across this session's M45 tests plus any pre-M45 tests touching `material.Appearance`.

## PBI breakdown (M46)

Mirrors M45's F01–F05 shape:

- **F01 (API)** — delete legacy contract/wire/client; rename `OpenPBRAppearance`→
  `Appearance` throughout. One PBI (small repo, one coordinated rename+deletion).
  Version bump to `v0.152.0`.
- **F02 (GPL model layer)** — `AssignmentStore`, `Library`/`AssetSet`/`MergedLookup`
  consolidation + renames; delete `migrateAppearance` + its test.
- **F03 (Catalog migration)** — rewrite the 93 catalog YAML files via a throwaway
  one-time conversion script, hand-reviewed on a sample, script then deleted.
- **F04 (Persistence)** — `recipe.go` field renames + the one-time legacy-shape
  migration, tested against a real pre-migration `.obk` and a real pre-migration
  project-library YAML fixture.
- **F05 (Router + UI)** — delete the legacy router handlers/editor tab, promote the
  OpenPBR editor to the sole tab; update every existing test referencing old names.
- **F06 (Live verification + cleanup)** — MCP-driven live test proving appearance
  assignment/rendering still works post-rename (same technique as M45 PBI-354), full
  suite + lint, then open the new `Oblikovati.API` PR and push the accumulated commits
  on `feat/m45-openpbr-host` (extending GPL PR#2152).

## Out of scope

- Anything not already covered by M45's OpenPBR lobe set (coat/fuzz/subsurface/
  transmission live-shader porting stays tracked separately as #2148).
- Any change to how Realistic mode resolves a `Surface` from an `Appearance` beyond
  the rename (the `openPBRAppearanceSurface`/`linearBaseColor` fixes from M45 PBI-354,
  #2150, are unaffected — they operate on the now-sole `Appearance` type unchanged).
- `capture_viewport`'s stale-render-target gap (#2149) — unrelated, separately tracked.
- The pre-existing, unrelated `head/ui` lint debt (#2151) — unrelated, separately
  tracked; may incidentally shrink since `appearance_editor.go` (one of its two
  flagged files) is being deleted, but that's a side effect, not a goal here.

## Verification

- `go build ./...` + `go vet ./...` across both modules (root `Oblikovati` +
  `Oblikovati.API` via `go.work`).
- `go test ./...` — every renamed/relocated function keeps or gains a test per
  CLAUDE.md's "every new function gets a test."
- `golangci-lint run` clean on every touched file (the `.golangci.yml` SA1019
  exclusions added for the now-deleted legacy `Appearance` become unnecessary and get
  removed as part of F02/F05).
- Old-format fixture round-trip tests (F04) prove the one-time migration actually
  converts real old data, not just synthetic in-test structs.
- Live `mcplive`/MCP-bridge verification (F06) — same technique M45 PBI-354 used —
  before either PR opens, per this repo's live-test convention.
