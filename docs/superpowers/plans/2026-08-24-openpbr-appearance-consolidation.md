# OpenPBR Appearance Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete the legacy metallic-roughness `Appearance` material system, rename
`OpenPBRAppearance` to `Appearance` everywhere (types/contract/wire/client, GPL model,
router, UI), migrate the built-in catalog to the native OpenPBR schema, and add a
one-time load-time migration for old `.obk`/project-library data — so OpenPBR Surface
becomes the sole material representation instead of a parallel addition.

**Architecture:** Two repos, sequenced by a hard dependency: `Oblikovati.API`'s rename
must merge and release a new tag FIRST (F01), because the GPL repo (`Oblikovati`)
cannot compile against the renamed contract until that tag exists and its `go.mod` pin
is bumped to it. GPL work (F02-F05) then proceeds on the SAME branch
(`feat/m45-openpbr-host`) and SAME open PR (#2152) that M45 already built.

**Tech Stack:** Go 1.26, `go.work` for local cross-repo dev, YAML persistence
(`gopkg.in/yaml.v3` via `oblikovati.org/yamlcodec`), `golangci-lint` v2.12.2.

**Spec:** `docs/superpowers/specs/2026-08-24-openpbr-appearance-consolidation-design.md`

## Global Constraints

- No shipped add-ins depend on the legacy wire API — breaking changes are acceptable
  (confirmed with the user during brainstorming).
- `go build ./...` + `go vet ./...` + `go test ./...` + `golangci-lint run ./...` must
  stay clean after every task, in BOTH repos once F01 has released.
- `contract.Material.AppearanceID()` does not change signature — only what its string
  id resolves through.
- The 9 OpenPBR group types (`types.OpenPBRBase`, `OpenPBRSpecular`, `OpenPBRCoat`,
  `OpenPBRFuzz`, `OpenPBRThinFilm`, `OpenPBRTransmission`, `OpenPBRSubsurface`,
  `OpenPBREmission`, `OpenPBRGeometry`) keep their `OpenPBR` prefix — only the
  top-level container types, DTOs, method names, and MCP tool names lose it.
- Every renamed/relocated function keeps or gains a test (CLAUDE.md: "every new
  function gets a test").
- `Oblikovati.API` repo path: `/home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati.API`.
  `Oblikovati` repo path: `/home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati`.

---

## F01 — Oblikovati.API: delete legacy, rename OpenPBRAppearance → Appearance

### Task 1: New branch, delete the legacy Appearance contract/wire/client

**Files:**
- Modify: `Oblikovati.API` — new branch `feat/appearance-consolidation` off `develop`.
- Delete: `contract/appearance.go` (the `Appearance` interface).
- Modify: `wire/material.go` — delete `AppearanceInfo`, `ListAppearancesResult`,
  `AssignAppearanceArgs` types. KEEP `AssetRefArgs`, `DuplicateAssetArgs` (shared with
  Materials), `MaterialInfo`, `ListMaterialsResult`, `AssignMaterialArgs`.
- Modify: `wire/methods.go` — delete `MethodAppearancesList`/`Get`/`Create`/`Update`
  (currently `wire/methods.go:702-705`) and `MethodModelAssignAppearance` (currently
  `wire/methods.go:720`). Keep `MethodMaterials*` and `MethodModelAssignMaterial`.
- Delete: `client/appearances.go` (the `Appearances` method group).
- Delete: `types/openpbr_surface.go`'s `OpenPBRSurfaceParams` type and
  `DefaultOpenPBRSurfaceParams` function — confirmed dead code (grep shows zero
  references outside their own declaration file); do not carry it into the rename.

**Interfaces:**
- Produces: nothing new — this task only deletes. `contract.OpenPBRAppearance`,
  `wire.OpenPBRAppearanceInfo`, `client.OpenPBRAppearances`, etc. still exist under
  their current names after this task (renamed in Task 2).

- [ ] **Step 1: Create the branch**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati.API
git checkout -b feat/appearance-consolidation origin/develop
```

- [ ] **Step 2: Delete `contract/appearance.go`**

```bash
rm contract/appearance.go
```

- [ ] **Step 3: Remove the legacy types from `wire/material.go`**

Edit `wire/material.go`: delete the `AppearanceInfo` struct, the
`ListAppearancesResult` struct, and the `AssignAppearanceArgs` struct. Leave
`MaterialInfo`, `ListMaterialsResult`, `AssetRefArgs`, `DuplicateAssetArgs`,
`AssignMaterialArgs` untouched. Update the file's remaining doc comments that
reference "PBR appearance" if they now read oddly next to only Material types (check
`MaterialInfo`'s comment doesn't cross-reference the deleted `AppearanceInfo`).

- [ ] **Step 4: Remove the legacy method constants from `wire/methods.go`**

Delete these four lines (currently at `wire/methods.go:702-705`):
```go
	MethodAppearancesList   = "appearances.list"
	MethodAppearancesGet    = "appearances.get"
	MethodAppearancesCreate = "appearances.create"
	MethodAppearancesUpdate = "appearances.update"
```
And this line (currently `wire/methods.go:720`):
```go
	MethodModelAssignAppearance        = "model.assignAppearance"
```
Leave `MethodMaterialsList`/`Get`/`Create`/`Update` and
`MethodModelAssignMaterial` in place.

- [ ] **Step 5: Delete `client/appearances.go`**

```bash
rm client/appearances.go
```

- [ ] **Step 6: Delete the dead `OpenPBRSurfaceParams` type**

Edit `types/openpbr_surface.go`: delete the `OpenPBRSurfaceParams` struct and
`DefaultOpenPBRSurfaceParams` function (the whole file's content, since it contains
nothing else) — but do NOT delete the file yet, Task 2 repurposes it.

- [ ] **Step 7: Build and fix fallout**

```bash
GOWORK=off go build ./... 2>&1
```
Expected: compile errors in any file still referencing `contract.Appearance`,
`wire.AppearanceInfo`, `wire.ListAppearancesResult`, `wire.AssignAppearanceArgs`,
`wire.MethodAppearances*`, `wire.MethodModelAssignAppearance`, or
`client.Appearances`/`client.OpenPBRAppearances().Assign` calling the deleted
constant. Fix each by deleting the reference (there should be none outside tests,
since nothing in this repo's own non-test code consumed the legacy client group —
verify with `grep -rn "client.Appearances\b" --include="*.go" .` before assuming).

- [ ] **Step 8: Delete/update tests referencing the deleted symbols**

```bash
grep -rln "contract.Appearance\b\|wire.AppearanceInfo\|wire.AssignAppearanceArgs\|wire.ListAppearancesResult\|wire.MethodAppearances\|wire.MethodModelAssignAppearance\|client.Appearances\b" --include="*_test.go" .
```
For each file found, delete the specific test cases exercising the deleted legacy
API (not the whole file unless it becomes empty) — the OpenPBR-named equivalents in
the same files stay untouched by this task (Task 2 renames them).

- [ ] **Step 9: Run the full test suite**

```bash
GOWORK=off go test ./... 2>&1
```
Expected: PASS. Fix any remaining fallout before continuing.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
Delete legacy Appearance contract/wire/client (M46-F01)

Removes contract.Appearance, the appearance.* wire DTOs/methods, and
client.Appearances — the metallic-roughness material system, superseded
by OpenPBRAppearance (renamed to Appearance next). Also deletes the
orphaned types.OpenPBRSurfaceParams (zero references anywhere).

No shipped add-in depends on the legacy wire API yet, so this is a
deliberate breaking change (confirmed with the user) rather than a
deprecate-and-keep.
EOF
)"
```

### Task 2: Rename OpenPBRAppearance → Appearance across types/contract/wire/client

**Files:**
- Modify: `contract/openpbr_appearance.go` → rename file to `contract/appearance.go`
  (reclaiming the name Task 1 freed); rename the `OpenPBRAppearance` interface to
  `Appearance`.
- Modify: `wire/openpbr_appearance.go` → rename file to `wire/appearance.go`; rename
  `OpenPBRAppearanceInfo`→`AppearanceInfo`, `ListOpenPBRAppearancesResult`→
  `ListAppearancesResult`, `CreateOpenPBRAppearanceArgs`→`CreateAppearanceArgs`,
  `UpdateOpenPBRAppearanceArgs`→`UpdateAppearanceArgs`,
  `AssignOpenPBRAppearanceArgs`→`AssignAppearanceArgs`.
- Modify: `wire/methods.go` — rename `MethodOpenPBRAppearancesList`/`Get`/`Create`/
  `Update`→`MethodAppearancesList`/`Get`/`Create`/`Update` (values
  `"openpbrAppearances.list"`→`"appearances.list"` etc.), rename
  `MethodModelAssignOpenPBRAppearance`→`MethodModelAssignAppearance` (value
  `"model.assignOpenPBRAppearance"`→`"model.assignAppearance"`). Delete the now-stale
  doc comment above them referencing "additive alongside appearances.*".
- Modify: `client/openpbr_appearances.go` → rename file to `client/appearances.go`;
  rename `OpenPBRAppearances` struct/method→`Appearances`, and every MCP tool name in
  its doc comments: `list_openpbr_appearances`→`list_appearances`,
  `get_openpbr_appearance`→`get_appearance`, `create_openpbr_appearance`→
  `create_appearance`, `update_openpbr_appearance`→`update_appearance`,
  `assign_openpbr_appearance`→`assign_appearance`. Rename the `mcp:digest
  summarizeOpenPBRAppearances`→`summarizeAppearances` reference too (the digest
  function itself lives in the MCP bridge repo, out of scope here — this only renames
  what THIS repo's annotation says).
- Modify: `model/material` is GPL, not touched in this task (F02).
- Delete: `types/openpbr_surface.go` (now empty after Task 1's Step 6 — confirm and
  remove the file entirely).
- Modify: any file defining `DefaultOpenPBRAppearanceID`-equivalent — NONE exist in
  this repo (that constant is GPL-side, `model/material/openpbr_builtin.go`, handled
  in F02). Nothing to do here.

**Interfaces:**
- Consumes: nothing from Task 1 beyond the deletions already landed.
- Produces: `contract.Appearance` (interface: `ID() string`, `DisplayName() string`,
  `Source() types.AssetSource`, `Base() types.OpenPBRBase`, `Specular()
  types.OpenPBRSpecular`, `Transmission() types.OpenPBRTransmission`, `Subsurface()
  types.OpenPBRSubsurface`, `Coat() types.OpenPBRCoat`, `Fuzz() types.OpenPBRFuzz`,
  `ThinFilm() types.OpenPBRThinFilm`, `Emission() types.OpenPBREmission`, `Geometry()
  types.OpenPBRGeometry`); `wire.AppearanceInfo` (JSON DTO, all 9 groups plus
  id/displayName/source); `wire.MethodAppearancesList/Get/Create/Update` =
  `"appearances.list"`/`"appearances.get"`/`"appearances.create"`/
  `"appearances.update"`; `wire.MethodModelAssignAppearance` =
  `"model.assignAppearance"`; `client.Appearances` method group with `List() (wire.
  ListAppearancesResult, error)`, `Get(id string) (wire.AppearanceInfo, error)`,
  `Create(wire.DuplicateAssetArgs) (wire.AppearanceInfo, error)`, `Update(wire.
  UpdateAppearanceArgs) (wire.AppearanceInfo, error)`, `Assign(wire.
  AssignAppearanceArgs) (wire.OKResult, error)`.

- [ ] **Step 1: Rename and edit `contract/openpbr_appearance.go`**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati.API
git mv contract/openpbr_appearance.go contract/appearance.go
```
Edit the moved file: rename `type OpenPBRAppearance interface` to `type Appearance
interface`. Update its doc comment to drop the "additive alongside Appearance, see
Appearance's deprecation note" framing (that appearance no longer exists) — replace
with a comment describing it as the sole appearance contract, covering the full
OpenPBR Surface v1.1.1 spec.

- [ ] **Step 2: Rename and edit `wire/openpbr_appearance.go`**

```bash
git mv wire/openpbr_appearance.go wire/appearance.go
```
In the moved file, rename every type per the Files section above
(`OpenPBRAppearanceInfo`→`AppearanceInfo`, etc.). Update doc comments that say
"additive alongside AppearanceInfo's metallic-roughness subset" — that subset is
gone; describe it as the sole appearance DTO instead.

- [ ] **Step 3: Rename the method constants in `wire/methods.go`**

Replace the block currently at `wire/methods.go:714-721`:
```go
	// OpenPBR Surface appearances (M45-F02, Oblikovati#2124): additive alongside
	// appearances.* — the full OpenPBR lobe set instead of metallic-roughness.
	MethodOpenPBRAppearancesList   = "openpbrAppearances.list"
	MethodOpenPBRAppearancesGet    = "openpbrAppearances.get"
	MethodOpenPBRAppearancesCreate = "openpbrAppearances.create"
	MethodOpenPBRAppearancesUpdate = "openpbrAppearances.update"

	MethodModelAssignMaterial          = "model.assignMaterial"
	MethodModelAssignOpenPBRAppearance = "model.assignOpenPBRAppearance"
```
with:
```go
	MethodAppearancesList   = "appearances.list"
	MethodAppearancesGet    = "appearances.get"
	MethodAppearancesCreate = "appearances.create"
	MethodAppearancesUpdate = "appearances.update"

	MethodModelAssignMaterial   = "model.assignMaterial"
	MethodModelAssignAppearance = "model.assignAppearance"
```
(`MethodMaterialsList`/`Get`/`Create`/`Update` stay where they are, just above.)

- [ ] **Step 4: Rename and edit `client/openpbr_appearances.go`**

```bash
git mv client/openpbr_appearances.go client/appearances.go
```
In the moved file, rename `OpenPBRAppearances` struct → `Appearances`, its
constructor method `(c *Client) OpenPBRAppearances() OpenPBRAppearances` →
`(c *Client) Appearances() Appearances`, every receiver type, and every `mcp:tool`/
`mcp:digest` annotation per the Files section above. Update method bodies to call the
renamed `wire.Method*`/`wire.*Args`/`wire.*Info` symbols from Steps 2-3.

- [ ] **Step 5: Delete the now-empty `types/openpbr_surface.go`**

```bash
rm types/openpbr_surface.go
```

- [ ] **Step 6: Build**

```bash
GOWORK=off go build ./... 2>&1
```
Fix any remaining references to the old names (search
`grep -rn "OpenPBRAppearance\b\|OpenPBRAppearances\b" --include="*.go" . | grep -v "OpenPBRAppearanceInfo\|types\.OpenPBR[A-Z]"` —
that pattern should return nothing once done, since only the group types keep an
`OpenPBR` prefix).

- [ ] **Step 7: Update tests**

Rename every `Test*OpenPBRAppearance*` test function and every reference to the old
symbol names in `*_test.go` files across `contract/`, `wire/`, `client/`, `types/`.
Run:
```bash
GOWORK=off go test ./... 2>&1
```
Expected: PASS.

- [ ] **Step 8: Lint**

```bash
GOWORK=off golangci-lint run ./... 2>&1
```
Expected: 0 issues.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
Rename OpenPBRAppearance to Appearance (M46-F01)

The plain name is free now that the legacy metallic-roughness
Appearance is deleted. Renames the top-level container types, wire
DTOs, wire method constants, the client method group, and MCP tool
names. The nine OpenPBR group types (types.OpenPBRBase,
OpenPBRSpecular, etc.) keep their prefix — bare names would be
dangerously ambiguous in this package.

Also deletes types.OpenPBRSurfaceParams, dead code with zero
references anywhere, discovered while auditing this rename's scope.
EOF
)"
```

### Task 3: Open the API PR, merge on green, verify the release tag

**Files:** none (process task).

**Interfaces:**
- Consumes: Task 2's commit.
- Produces: a new `vX.Y.0` git tag on `Oblikovati.API` (the auto-release workflow
  derives the exact version from commit scope — this is a breaking change to a
  pre-1.0 package, which per SemVer's own pre-1.0 convention bumps MINOR, so expect
  `v0.151.0`→`v0.152.0`, but confirm against whatever the release workflow actually
  computes rather than assuming).

- [ ] **Step 1: Push and open the PR**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati.API
git push -u origin feat/appearance-consolidation
gh pr create --title "Consolidate OpenPBRAppearance as the sole appearance contract" --body "$(cat <<'EOF'
## Summary
- Deletes the legacy metallic-roughness Appearance contract/wire/client (PR#282
  landed it as additive; this removes it per the M46 consolidation decision).
- Renames OpenPBRAppearance -> Appearance now that the name is free.
- No shipped add-in depends on the legacy wire API yet, so this breaking change is
  acceptable (confirmed with the maintainer).

## Test plan
- [x] `GOWORK=off go build ./...`
- [x] `GOWORK=off go vet ./...`
- [x] `GOWORK=off go test ./...`
- [x] `GOWORK=off golangci-lint run ./...`

Spec: docs/superpowers/specs/2026-08-24-openpbr-appearance-consolidation-design.md
(in the companion Oblikovati repo).
EOF
)"
```

- [ ] **Step 2: Watch CI, merge when green**

```bash
gh pr checks --watch
gh pr merge --merge --delete-branch=false
```

- [ ] **Step 3: Confirm the new release tag**

```bash
git fetch --tags origin
git tag --sort=-creatordate | head -3
```
Record the new tag (e.g. `v0.152.0`) — Task 4 needs it to pin the GPL repo.

---

## F02 — GPL model layer: single chain, rename, delete legacy

### Task 4: Pin `Oblikovati` to the new API release

**Files:**
- Modify: `go.mod`, `head/go.mod` (in `Oblikovati`, NOT `.API`).

**Interfaces:**
- Consumes: the tag from F01 Task 3, Step 3.
- Produces: both modules requiring the new `oblikovati.org/api` version.

- [ ] **Step 1: Bump the pin**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati
scripts/sync-api-version.sh
```

- [ ] **Step 2: Build — expect failure**

```bash
go build ./... 2>&1
```
Expected: FAIL, with compile errors everywhere `material.Appearance`,
`contract.Appearance` (via the `var _ contract.Appearance = (*Appearance)(nil)`
assertion), or the OLD `contract.OpenPBRAppearance`/`wire.OpenPBRAppearanceInfo`
names are referenced — this is expected and confirms the pin took effect; Tasks 5-8
fix it.

- [ ] **Step 3: Commit the pin bump on its own**

```bash
git add go.mod head/go.mod
git commit -m "build: pin oblikovati.org/api to the appearance-consolidation release (M46-F02)"
```

### Task 5: `AssignmentStore` — collapse to a single chain

**Files:**
- Modify: `model/material/assignment.go`.
- Modify: `model/material/assignment_test.go`.

**Interfaces:**
- Consumes: nothing new (pure rename/deletion of existing GPL code).
- Produces: `AssignmentStore.SetPartAppearance(id string)`,
  `SetBodyAppearance(key, id string)`, `SetFaceAppearance(key, id string)`,
  `PartAppearance() string`, `BodyAppearances() map[string]string`,
  `FaceAppearances() map[string]string`, `EffectiveAppearance(look AssetLookup,
  bodyKey, faceKey string) *Appearance` (always non-nil, `look.DefaultAppearance()`
  fallback — same contract the OLD `EffectiveAppearance` had). `AssetLookup` interface
  now has exactly: `Appearance(id string) (*Appearance, bool)`, `Material(id string)
  (*Material, bool)`, `DefaultAppearance() *Appearance`.

- [ ] **Step 1: Rewrite `assignment.go`**

Replace the `AssignmentStore` struct's fields:
```go
type AssignmentStore struct {
	partMaterial   string
	partAppearance string
	bodyMaterial   map[string]string // bodyKey → material id
	bodyAppearance map[string]string // bodyKey → appearance id
	faceAppearance map[string]string // faceKey → appearance id
}
```
(deleting the `partOpenPBRAppearance`/`bodyOpenPBRAppearance`/
`faceOpenPBRAppearance` fields and their doc comment entirely — the struct goes back
to exactly its pre-M45 shape).

`NewAssignmentStore` drops the `bodyOpenPBRAppearance`/`faceOpenPBRAppearance` map
initialization.

Delete `SetPartOpenPBRAppearance`/`SetBodyOpenPBRAppearance`/
`SetFaceOpenPBRAppearance`, `PartOpenPBRAppearance`/`BodyOpenPBRAppearances`/
`FaceOpenPBRAppearances`, `EffectiveOpenPBRAppearance`, `openPBRApprOrNil`.

Replace `AssetLookup`:
```go
type AssetLookup interface {
	Appearance(id string) (*Appearance, bool)
	Material(id string) (*Material, bool)
	DefaultAppearance() *Appearance
}
```
(drop the `OpenPBRAppearance(id string) (*OpenPBRAppearance, bool)` method — Task 6
renames `*OpenPBRAppearance` types to `*Appearance`, at which point this interface's
remaining `Appearance`/`DefaultAppearance` methods already cover it; there is no
separate OpenPBR lookup method anymore).

Replace `EffectiveAppearance` (delete the OLD one that reads `s.faceAppearance`/
`s.bodyAppearance`/`s.partAppearance`/`s.EffectiveMaterial`-chain — that logic is
gone since Material→Appearance fallback via `m.AppearanceID()` still applies, so
actually KEEP that fallback chain shape, just rename the field reads if needed — they
already match). Concretely: keep the CURRENT `EffectiveAppearance` body verbatim
(it already implements exactly the face→body→part→material→default precedence this
task wants), and additionally rename `apprOrNil`'s signature is unchanged. Then ADD
nothing else — `EffectiveOpenPBRAppearance`'s deletion means `EffectiveAppearance`
is once again the ONLY resolver, so no further change to its body is needed beyond
what already exists.

- [ ] **Step 2: Update `assignment_test.go`**

Delete `TestEffectiveOpenPBRAppearancePrecedence` (the dual-chain test added this
session — its coverage is subsumed by keeping `TestEffectiveAppearancePrecedence`,
which already tests the exact same face→body→part precedence now that there's only
one chain). Verify `TestEffectiveAppearancePrecedence`,
`TestPartAppearanceOverrideBeatsMaterialAppearance`, `TestEffectiveMaterialOverride`,
`TestEffectiveMaterialID`, `TestSetEmptyClearsAssignment` all still compile as-is
(they reference `NewAssignmentStore`/`SetPartMaterial`/`SetBodyMaterial`/
`SetBodyAppearance`/`SetFaceAppearance`/`SetPartAppearance`, none of which changed
shape).

- [ ] **Step 3: Run the package tests**

```bash
go test ./model/material/... 2>&1
```
Expected: still FAIL (Task 6 hasn't renamed `OpenPBRAppearance`→`Appearance` yet, so
`Library`/`AssetSet` won't satisfy the new `AssetLookup` shape). Confirm the failures
are ONLY in files Task 6 will touch (`library.go`, `openpbr_library.go`,
`assetset.go`, and their tests) — no failures in `assignment.go`/
`assignment_test.go` themselves.

- [ ] **Step 4: Commit**

```bash
git add model/material/assignment.go model/material/assignment_test.go
git commit -m "AssignmentStore: collapse to a single appearance chain (M46-F02)"
```

### Task 6: `Library`/`AssetSet`/`MergedLookup` — delete legacy, rename OpenPBR

**Files:**
- Modify: `model/material/library.go` — delete legacy `Appearance`-typed methods,
  keep `Material`-typed ones.
- Modify: `model/material/openpbr_library.go` → rename to
  `model/material/library_appearances.go`; rename every `OpenPBRAppearance`-suffixed
  method to drop the prefix.
- Modify: `model/material/assetset.go` — delete the legacy `appearances` map/methods,
  rename `openpbrAppearances`→`appearances`, rename every `OpenPBRAppearance`-suffixed
  method.
- Modify: `model/material/appearance.go` — DELETE this file (the legacy `Appearance`
  struct/`AppearanceSpec`).
- Modify: `model/material/openpbr_appearance.go` → rename to
  `model/material/appearance.go` (reclaiming the name the delete just freed); rename
  `OpenPBRAppearance`→`Appearance`, `OpenPBRAppearanceSpec`→`AppearanceSpec`.
- Modify: `model/material/openpbr_builtin.go` → rename to `model/material/builtin_appearance.go`;
  rename `DefaultOpenPBRAppearanceID`→`DefaultAppearanceID` (value `"openpbr-default"`
  → `"default"`), `defaultOpenPBRAppearance`→`defaultAppearance`.
- Modify: `model/material/builtin.go` — DELETE the current `DefaultAppearanceID`
  const, `defaultAppearance` function, and `mustColor` helper's legacy-only
  callers if any (keep `mustColor` itself — check if `builtin_appearance.go`'s new
  content needs it; the OpenPBR default builds colors via `Color3` literals, not hex,
  so `mustColor` likely becomes genuinely dead — grep for other callers before
  deleting it).
- Modify: `model/material/catalog.go` — `loadCatalogFile`'s appearance-loading loop
  changes shape (Task 8 handles the catalog-format side; this task only needs
  `l.AddAppearance`/`l.AddOpenPBRAppearance` calls updated to the renamed single
  `l.AddAppearance` — but doing so BEFORE Task 8 rewrites the YAML shape would break
  catalog loading, since `rd.Appearances`/`recipeToAppearance` still parse the OLD
  5-scalar recipe shape. Resolve by leaving `loadCatalogFile` referencing
  `rd.Appearances`/`recipeToAppearance` UNCHANGED in this task — Task 9 (catalog
  rewrite) and its own recipe-shape change happen together, not here).
- Modify: `model/material/*_test.go` for every file above.

**Interfaces:**
- Consumes: `AssignmentStore.AssetLookup` interface from Task 5 (exactly
  `Appearance`/`Material`/`DefaultAppearance`).
- Produces: `Library.Appearances() []*Appearance`, `Library.Appearance(id string)
  (*Appearance, bool)`, `Library.DefaultAppearance() *Appearance`,
  `Library.AddAppearance(*Appearance)`, `Library.DuplicateAppearance(baseID, name
  string, source Source) (*Appearance, error)`, `Library.EditAppearance(id string,
  spec AppearanceSpec)`, `Library.RemoveAppearance(id string) error`. Same set on
  `AssetSet` (`PutAppearance`, `Appearance`, `Appearances`) and `MergedLookup`
  (`Appearance`, `DefaultAppearance`). `DefaultAppearanceID = "default"`.

- [ ] **Step 1: Delete the legacy struct, promote the OpenPBR one**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati
rm model/material/appearance.go
git mv model/material/openpbr_appearance.go model/material/appearance.go
```
In the moved file: rename `type OpenPBRAppearanceSpec struct` → `type AppearanceSpec
struct` (this REPLACES the just-deleted `AppearanceSpec` — same name, new 9-group
shape); rename `type OpenPBRAppearance struct` → `type Appearance struct`; rename
`NewOpenPBRAppearance`→`NewAppearance`; update the `var _ contract.Appearance =
(*Appearance)(nil)` assertion (it already says `contract.Appearance` per the earlier
`var _ contract.OpenPBRAppearance = (*OpenPBRAppearance)(nil)` line — rename both
sides); update every method receiver/return type
(`Base()`/`Specular()`/etc. keep their names, just now methods of `*Appearance`).

- [ ] **Step 2: Rename `library.go`'s Appearance methods to work with the new type**

`library.go`'s existing `Appearances()`/`Appearance(id)`/`DefaultAppearance()`/
`AddAppearance`/`DuplicateAppearance`/`EditAppearance`/`RemoveAppearance` already
have the right NAMES (they were the legacy ones) — change every `*Appearance` type
reference in their bodies/signatures to point at the now-9-group `Appearance` struct
from Step 1 (no name changes needed in this file, since Step 1 made `*Appearance`
BE the OpenPBR-shaped type — this "just works" as long as the struct rename in Step
1 landed first). Verify `l.appearances map[string]*Appearance` in the `Library`
struct now holds 9-group appearances.

Delete the OLD `l.openpbrAppearances map[string]*OpenPBRAppearance` field and
`l.openpbrApprOrder []string` field from the `Library` struct (declared in
`library.go`) — the renamed `l.appearances`/`l.apprOrder` from Step 2 IS what used
to be `openpbrAppearances`/`openpbrApprOrder`... **resolve the two colliding fields**
by DELETING `library.go`'s original `appearances`/`apprOrder` fields (which held the
now-deleted 5-scalar struct) and RENAMING `openpbrAppearances`→`appearances`,
`openpbrApprOrder`→`apprOrder` (declared today in `library.go`'s struct literal, not
`openpbr_library.go`). Concretely: in `library.go`'s `Library` struct and
`NewLibrary()`, delete the two `appearances`/`apprOrder` field declarations/inits
that held the legacy type, then rename `openpbrAppearances`→`appearances` and
`openpbrApprOrder`→`apprOrder` throughout the file (both the struct fields and every
method body in `library.go` that references them, i.e. `Appearances()`,
`Appearance(id)`, `DefaultAppearance()`, `AddAppearance()`, `DuplicateAppearance()`,
`EditAppearance()`, `RemoveAppearance()` all now read/write what used to be the
`openpbr*`-prefixed fields).

- [ ] **Step 3: Delete `openpbr_library.go`'s now-duplicate methods**

Since Step 2 renamed `library.go`'s own methods to operate on the 9-group type
directly, `openpbr_library.go`'s methods
(`OpenPBRAppearances`/`OpenPBRAppearance`/`DefaultOpenPBRAppearance`/
`AddOpenPBRAppearance`/`DuplicateOpenPBRAppearance`/`EditOpenPBRAppearance`/
`RemoveOpenPBRAppearance`) are now REDUNDANT with `library.go`'s renamed ones —
delete the file entirely:
```bash
rm model/material/openpbr_library.go
```

- [ ] **Step 4: `assetset.go`**

Delete the struct's `appearances map[string]*Appearance` field (legacy) and its
`PutAppearance`/`Appearance(id)`/`Appearances()` methods that operate on it. Rename
`openpbrAppearances`→`appearances`, and rename
`PutOpenPBRAppearance`→`PutAppearance`, `OpenPBRAppearance(id)`→`Appearance(id)`,
`OpenPBRAppearances()`→`Appearances()`. Same for `MergedLookup`: delete the legacy
`Appearance`/method bodies that read `m.Catalog.Appearance`/`m.Embedded.Appearance`
against the OLD type (they already have the right NAMES — just verify their bodies
now resolve through the Step 2/3 renamed `Library`/`AssetSet` methods, and delete
the separate `OpenPBRAppearance` method on `MergedLookup` since it's now redundant
with the renamed `Appearance` method).

- [ ] **Step 5: `builtin.go` / `openpbr_builtin.go`**

```bash
git mv model/material/openpbr_builtin.go model/material/builtin_appearance.go
```
In `builtin.go`: delete `DefaultAppearanceID` const and `defaultAppearance()`
function (the legacy 5-scalar one). In `builtin_appearance.go`: rename
`DefaultOpenPBRAppearanceID`→`DefaultAppearanceID`, change its value from
`"openpbr-default"` to `"default"`; rename `defaultOpenPBRAppearance`→
`defaultAppearance`; update its doc comment (drop the "OpenPBR-side counterpart of
[DefaultAppearanceID]" framing — it's the only one now); its body already builds
`AppearanceSpec` (post Step 1 rename) so no further change needed beyond the
renames. In `builtin.go`'s `seedBuiltins`, change:
```go
	l.AddAppearance(defaultAppearance())
	l.AddOpenPBRAppearance(defaultOpenPBRAppearance())
```
to:
```go
	l.AddAppearance(defaultAppearance())
```
(single call now — `defaultAppearance` in `builtin_appearance.go` shadows the
just-deleted one in `builtin.go` after Step 5's edits, so this is the ONE remaining
call site). Check whether `mustColor` (in `builtin.go`) has any remaining callers
after this deletion:
```bash
grep -rn "mustColor(" --include="*.go" model/material/
```
If the only remaining callers are in `*_test.go` files, leave `mustColor` in place
(tests still use it); if zero callers remain anywhere, delete it.

- [ ] **Step 6: Build and fix fallout**

```bash
go build ./model/material/... 2>&1
```
Fix compile errors mechanically — every remaining reference to `OpenPBRAppearance`/
`OpenPBRAppearanceSpec`/`OpenPBRAppearance(id)`/etc. outside the group types gets
renamed per the pattern established in Steps 1-5.

- [ ] **Step 7: Update tests**

Merge `model/material/openpbr_library_test.go` into `model/material/library_test.go`
(delete the `_test.go` for the file deleted in Step 3, move its still-relevant test
cases — updated to the new names — into `library_test.go`, deleting any test that
now duplicates an existing `library_test.go` case testing the same renamed method).
Update `model/material/catalog_test.go`, `model/material/store_test.go`,
`model/material/openpbr_store_test.go` for the renamed symbols (do NOT yet touch
catalog-format-shape assertions — Task 9 handles those). Run:
```bash
go test ./model/material/... 2>&1
```
Expected: still some catalog-loading failures until Task 8/9 land (the recipe/YAML
shape mismatch) — confirm failures are SCOPED to catalog loading, not to the
Library/AssetSet API surface itself.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
model/material: delete legacy Appearance, rename OpenPBRAppearance (M46-F02)

Library/AssetSet/MergedLookup collapse to a single Appearance type (the
former OpenPBRAppearanceSpec/OpenPBRAppearance, now the sole material
representation). DefaultOpenPBRAppearanceID's value changes from
"openpbr-default" to "default", reclaiming the legacy id now that it's
free.
EOF
)"
```

### Task 7: `recipe.go`/`openpbr_recipe.go` — merge, rename persisted field names

**Files:**
- Modify: `model/material/recipe.go`.
- Delete: `model/material/openpbr_recipe.go` (merged into `recipe.go`).
- Modify: `model/material/recipe_test.go`.

**Interfaces:**
- Consumes: `Appearance`/`AppearanceSpec` from Task 6.
- Produces: `AppearanceRecipe` (the NEW 9-group YAML shape, reusing the name from
  `OpenPBRAppearanceRecipe`), `AssignmentRecipe.PartAppearance string`/
  `BodyAppearance map[string]string`/`FaceAppearance map[string]string` (single
  chain), `RecipeData.Appearances []AppearanceRecipe`, `MarshalRecipe`/`ApplyRecipe`
  unchanged signatures, `appearanceToRecipe(*Appearance) AppearanceRecipe`,
  `recipeToAppearance(AppearanceRecipe, Source) *Appearance` (NOTE: this function's
  signature changes — the OLD `recipeToAppearance` returned `(*Appearance, error)`
  for hex-parse failure; the NEW one, working on the 9-group shape with no hex
  colors, cannot fail on parsing, so it returns just `*Appearance`, no error — update
  every call site, including `catalog.go`'s `loadCatalogFile`, Task 4's error
  handling there goes away).

- [ ] **Step 1: Rewrite the recipe types**

In `recipe.go`, DELETE the old `AppearanceRecipe` struct (5-scalar). Move
`openpbr_recipe.go`'s `OpenPBRAppearanceRecipe` struct into `recipe.go`, renamed to
`AppearanceRecipe` (its shape is exactly the 9-group one this needs — `ID`,
`DisplayName`, `Base`, `Specular`, `Transmission`, `Subsurface`, `Coat`, `Fuzz`,
`ThinFilm`, `Emission`, `Geometry`, all with their current YAML tags unchanged).

In `AssignmentRecipe`, delete the `PartOpenPBRAppearance`/`BodyOpenPBRAppearance`/
`FaceOpenPBRAppearance` fields and their doc comment; the existing
`PartAppearance`/`BodyAppearance`/`FaceAppearance` fields (YAML tags
`partAppearance`/`bodyAppearance`/`faceAppearance`, unchanged) now carry the ids
that resolve through the single `Appearance` chain.

In `RecipeData`, delete the `OpenPBRAppearances []OpenPBRAppearanceRecipe` field;
the existing `Appearances []AppearanceRecipe` field now serializes 9-group data
(same YAML key `appearances`, unchanged — the ON-DISK key name for a document's own
embedded appearances doesn't change, only what's nested inside each entry).

- [ ] **Step 2: Rewrite the conversion functions**

Delete the OLD `appearanceToRecipe`/`recipeToAppearance` (hex-color, 5-scalar).
Move `openpbr_recipe.go`'s `openPBRAppearanceToRecipe`/`recipeToOpenPBRAppearance`
into `recipe.go`, renamed to `appearanceToRecipe`/`recipeToAppearance` — their
bodies already operate on `a.spec`/`OpenPBRAppearanceSpec`, which after Task 6 IS
`AppearanceSpec`, so only the function names and the `NewOpenPBRAppearance`→
`NewAppearance` call inside `recipeToOpenPBRAppearance`'s body need renaming.
`recipeToAppearance`'s new signature: `func recipeToAppearance(r AppearanceRecipe,
source Source) *Appearance` (no error return — confirm by re-reading the moved
body: it just calls `NewAppearance(r.ID, source, AppearanceSpec{...})` with no
fallible parsing, so `error` was never actually needed for this shape).

Delete `sortOpenPBRAppearances`; the existing `sortAppearances` (already present in
`recipe.go`, operating on `[]*Appearance`) covers it after Task 6's rename.

- [ ] **Step 3: Update `MarshalRecipe`/`ApplyRecipe`**

```go
func MarshalRecipe(set *AssetSet, assign *AssignmentStore) RecipeData {
	var data RecipeData
	for _, a := range sortAppearances(set.Appearances()) {
		data.Appearances = append(data.Appearances, appearanceToRecipe(a))
	}
	for _, m := range sortMaterials(set.Materials()) {
		data.Materials = append(data.Materials, materialToRecipe(m))
	}
	data.Assignments = assignmentRecipe(assign)
	return data
}

func ApplyRecipe(data RecipeData, set *AssetSet, assign *AssignmentStore) error {
	for _, ar := range data.Appearances {
		set.PutAppearance(recipeToAppearance(ar, SourceDocument))
	}
	for _, mr := range data.Materials {
		set.PutMaterial(recipeToMaterial(mr, SourceDocument))
	}
	if data.Assignments != nil {
		applyAssignmentRecipe(assign, data.Assignments)
	}
	return nil
}
```
(drops the `data.OpenPBRAppearances` loop entirely, and `recipeToAppearance` no
longer returns an error to check).

- [ ] **Step 4: Update `assignmentRecipe`/`applyAssignmentRecipe`**

```go
func assignmentRecipe(assign *AssignmentStore) *AssignmentRecipe {
	r := &AssignmentRecipe{
		PartMaterial:   assign.partMaterial,
		PartAppearance: assign.partAppearance,
		BodyMaterial:   nonEmpty(assign.bodyMaterial),
		BodyAppearance: nonEmpty(assign.bodyAppearance),
		FaceAppearance: nonEmpty(assign.faceAppearance),
	}
	if r.PartMaterial == "" && r.PartAppearance == "" &&
		r.BodyMaterial == nil && r.BodyAppearance == nil && r.FaceAppearance == nil {
		return nil
	}
	return r
}

func applyAssignmentRecipe(assign *AssignmentStore, r *AssignmentRecipe) {
	assign.partMaterial = r.PartMaterial
	assign.partAppearance = r.PartAppearance
	assign.bodyMaterial = orEmpty(r.BodyMaterial)
	assign.bodyAppearance = orEmpty(r.BodyAppearance)
	assign.faceAppearance = orEmpty(r.FaceAppearance)
}
```

- [ ] **Step 5: Delete `openpbr_recipe.go`**

```bash
rm model/material/openpbr_recipe.go
```

- [ ] **Step 6: Update `catalog.go`'s call site**

In `loadCatalogFile` (in `catalog.go`), the current:
```go
	for _, ar := range rd.Appearances {
		a, err := recipeToAppearance(ar, SourceBuiltin)
		if err != nil {
			return fmt.Errorf("material: catalog %q appearance %q: %w", name, ar.ID, err)
		}
		l.AddAppearance(a)
		l.AddOpenPBRAppearance(NewOpenPBRAppearance(a.ID(), SourceBuiltin, migrateAppearance(a)))
	}
```
becomes (no error to check, no migration call, no second `AddOpenPBRAppearance`):
```go
	for _, ar := range rd.Appearances {
		l.AddAppearance(recipeToAppearance(ar, SourceBuiltin))
	}
```
This makes `loadCatalogFile`'s own error return only reachable via the
`rd.Materials` loop now, if at all — leave the function signature as-is (`error`
return) since `yamlcodec.Unmarshal` above it can still fail.

- [ ] **Step 7: Build**

```bash
go build ./model/material/... 2>&1
```
Expected: still failing on `migrateAppearance` (Task 8 relocates it) and on
`catalog/*.yaml` parsing into the new 9-group `AppearanceRecipe` shape (Task 9
rewrites the YAML) — confirm remaining failures are scoped to exactly those two
things.

- [ ] **Step 8: Update `recipe_test.go`**

Rewrite tests exercising `AppearanceRecipe`/`appearanceToRecipe`/
`recipeToAppearance`/`MarshalRecipe`/`ApplyRecipe` for the 9-group shape (construct
a real `AppearanceSpec` with non-default `Base`/`Specular` values, round-trip it
through `appearanceToRecipe`→`recipeToAppearance`, assert equality). Delete any test
asserting the old hex-color parse-failure error path (no longer reachable).

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "recipe.go: merge OpenPBR appearance recipe, rename to reclaim plain names (M46-F02)"
```

### Task 8: Relocate the migration converter out of the live catalog-load path

**Files:**
- Modify: `model/material/openpbr_migrate.go` → rename to
  `model/material/legacy_migrate.go`.
- Modify: `model/material/openpbr_migrate_test.go` → rename to
  `model/material/legacy_migrate_test.go`.

**Interfaces:**
- Consumes: `AppearanceSpec` (9-group, from Task 6), a NEW type
  `legacyAppearanceSpec` representing the OLD 5-scalar shape (defined in this task —
  needed because Task 6 deleted the old `AppearanceSpec`, so the migration function
  needs its own input type now that nothing else in the codebase has the old shape).
- Produces: `legacyAppearanceToSpec(legacy legacyAppearanceSpec) AppearanceSpec` —
  Task 11 (F04) calls this from the one-time old-file migration path. NOT called
  from `catalog.go` anymore (Task 7 already removed that call site).

- [ ] **Step 1: Define the old-shape input type**

`migrateAppearance`'s current signature is `func migrateAppearance(a *Appearance)
OpenPBRAppearanceSpec` — it read `a.DisplayName()`/`a.Albedo()`/`a.Metallic()`/
`a.Roughness()`/`a.Emissive()`/`a.Opacity()` via the (now-deleted) legacy
`*Appearance` type's accessor methods. Since that type no longer exists, define a
plain struct carrying the same 6 fields the function actually reads:
```go
// legacyAppearanceSpec is the pre-M46 metallic-roughness appearance shape (fields
// mirror the deleted material.Appearance's accessors), used only by the one-time
// migration of old .obk/project-library data — never constructed from live code.
type legacyAppearanceSpec struct {
	DisplayName string
	Albedo      Rgba
	Metallic    float32
	Roughness   float32
	Emissive    Rgba
	Opacity     float32
}
```

- [ ] **Step 2: Rename and rewrite `migrateAppearance`**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati
git mv model/material/openpbr_migrate.go model/material/legacy_migrate.go
```
Rename `migrateAppearance`→`legacyAppearanceToSpec`, change its parameter from `a
*Appearance` to `a legacyAppearanceSpec`, and its body's field reads from
`a.DisplayName()`/`a.Albedo()`/etc. (method calls) to `a.DisplayName`/`a.Albedo`/etc.
(field reads):
```go
func legacyAppearanceToSpec(a legacyAppearanceSpec) AppearanceSpec {
	spec := DefaultOpenPBRAppearanceSpecForMigration()
	spec.DisplayName = a.DisplayName
	spec.Base.Color = srgbToLinearColor3(a.Albedo)
	spec.Base.Metalness = a.Metallic
	spec.Specular.Roughness = a.Roughness
	spec.Emission.Luminance, spec.Emission.Color = emissiveToLuminanceColor(a.Emissive)
	spec.Geometry.Opacity = a.Opacity
	return spec
}
```
Rename `DefaultOpenPBRAppearanceSpecForMigration`'s return type reference from
`OpenPBRAppearanceSpec` to `AppearanceSpec` (its body is otherwise unchanged —
`types.DefaultOpenPBRBase()`/`types.DefaultOpenPBRSpecular()`/
`OpenPBRGeometry{Opacity: 1}` calls stay as-is, those are the group types which kept
their prefix). Update its doc comment: it's no longer used by catalog loading
(that's gone), only by the migration test and Task 11's real migration path —
update the comment to say so.

Leave `srgbToLinearColor3`/`srgbToLinear`/`emissiveToLuminanceColor`/`maxOf3`
unchanged (pure helpers, no type dependency on the deleted `Appearance`).

- [ ] **Step 3: Build**

```bash
go build ./model/material/... 2>&1
```

- [ ] **Step 4: Rewrite the test**

```bash
git mv model/material/openpbr_migrate_test.go model/material/legacy_migrate_test.go
```
Update `TestMigrateAppearanceOffLobesAreZeroValue`-equivalent test to construct a
`legacyAppearanceSpec` literal directly (instead of `NewAppearance(...)`) and call
`legacyAppearanceToSpec`. `TestCatalogAppearancesHaveMigratedOpenPBRTwin` (the one
that iterated `lib.Appearances()` comparing catalog entries against migrated twins)
is now MEANINGLESS — there's only one appearance per catalog entry, not a
twin-comparison — DELETE this test entirely (Task 9's catalog-rewrite task adds its
own correctness test for the native YAML instead).

- [ ] **Step 5: Run and commit**

```bash
go test ./model/material/... 2>&1
git add -A
git commit -m "$(cat <<'EOF'
Relocate migrateAppearance to legacyAppearanceToSpec (M46-F02)

No longer runs at catalog-load time (the catalog is now natively
OpenPBR-authored, Task 9) — survives only for the one-time old-file
migration path (F04), operating on a small legacyAppearanceSpec struct
that stands in for the now-deleted Appearance type's old shape.
EOF
)"
```

---

## F03 — Catalog migration: rewrite the 7 YAML files to the native schema

### Task 9: One-time conversion script + hand-reviewed catalog rewrite

**Files:**
- Create (temporary): `model/material/cmd/convertcatalog/main.go` (deleted at the end
  of this task).
- Modify: all 7 files under `model/material/catalog/*.yaml`.
- Modify: `model/material/catalog_test.go`.

**Interfaces:**
- Consumes: `legacyAppearanceSpec`/`legacyAppearanceToSpec` from Task 8,
  `AppearanceRecipe`/`appearanceToRecipe` from Task 7.
- Produces: catalog YAML files whose `appearances:` entries are 9-group
  `AppearanceRecipe`-shaped instead of 5-scalar.

- [ ] **Step 1: Write the throwaway conversion script**

Create `model/material/cmd/convertcatalog/main.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only

// Command convertcatalog is a ONE-TIME, throwaway tool (M46-F03): reads a catalog
// YAML file's OLD 5-scalar appearance entries, converts each through the exact same
// mapping legacyAppearanceToSpec uses, and rewrites the file with 9-group entries in
// place. Deleted once every catalog file is converted and hand-reviewed — never a
// permanent part of the build.
//
//	go run ./model/material/cmd/convertcatalog catalog/01-metals.yaml
package main

import (
	"fmt"
	"os"

	"oblikovati.org/model/material"
	"oblikovati.org/yamlcodec"
)

// legacyRecipeData mirrors the OLD on-disk shape (pre-M46) so this throwaway tool can
// parse a not-yet-converted file without depending on the (already-deleted) old
// AppearanceRecipe/RecipeData types in the live material package.
type legacyRecipeData struct {
	Appearances []legacyAppearanceRecipe `yaml:"appearances,omitempty"`
	Materials   []map[string]any         `yaml:"materials,omitempty"` // passed through untouched
}

type legacyAppearanceRecipe struct {
	ID          string  `yaml:"id"`
	DisplayName string  `yaml:"name"`
	Albedo      string  `yaml:"albedo"`
	Metallic    float32 `yaml:"metallic"`
	Roughness   float32 `yaml:"roughness"`
	Emissive    string  `yaml:"emissive,omitempty"`
	Opacity     float32 `yaml:"opacity"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: convertcatalog <path/to/catalog-file.yaml>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "convertcatalog:", err)
		os.Exit(1)
	}
}

func run(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var old legacyRecipeData
	if err := yamlcodec.Unmarshal(data, &old); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	type newRecipeData struct {
		Appearances []material.AppearanceRecipe `yaml:"appearances,omitempty"`
		Materials   []map[string]any            `yaml:"materials,omitempty"`
	}
	var out newRecipeData
	out.Materials = old.Materials
	for _, a := range old.Appearances {
		spec := material.LegacyAppearanceToSpecForMigration(material.LegacyAppearanceSpecForMigration{
			DisplayName: a.DisplayName, AlbedoHex: a.Albedo, Metallic: a.Metallic,
			Roughness: a.Roughness, EmissiveHex: a.Emissive, Opacity: a.Opacity,
		})
		out.Appearances = append(out.Appearances, material.AppearanceRecipeForMigration(a.ID, spec))
	}

	encoded, err := yamlcodec.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}
```
This references two NOT-YET-EXISTING exported helpers
(`material.LegacyAppearanceSpecForMigration`, `material.LegacyAppearanceToSpecForMigration`,
`material.AppearanceRecipeForMigration`) because `legacyAppearanceSpec`/
`legacyAppearanceToSpec`/`appearanceToRecipe` from Tasks 7-8 are unexported
(package-private) and this script lives in a different package (`main`). Add a small
temporary exported shim in `model/material/legacy_migrate.go` (deleted alongside the
script at the end of this task):
```go
// LegacyAppearanceSpecForMigration / LegacyAppearanceToSpecForMigration /
// AppearanceRecipeForMigration are a temporary exported shim for
// cmd/convertcatalog's one-time catalog rewrite (M46-F03) — deleted together with
// that command once the catalog migration is done.
type LegacyAppearanceSpecForMigration = legacyAppearanceSpec

func LegacyAppearanceToSpecForMigration(a LegacyAppearanceSpecForMigration) AppearanceSpec {
	return legacyAppearanceToSpec(a)
}

func AppearanceRecipeForMigration(id string, spec AppearanceSpec) AppearanceRecipe {
	return appearanceToRecipe(NewAppearance(id, SourceBuiltin, spec))
}
```
The shim's `LegacyAppearanceSpecForMigration` fields must match what the script
constructs — since `legacyAppearanceSpec.Albedo`/`.Emissive` are `Rgba` (parsed), not
raw hex strings, either (a) change the shim to accept hex strings and parse them via
`types.ParseHex` before delegating, or (b) simplify: have the script call
`types.ParseHex` itself before building the shim struct. Prefer (b) — keeps the shim
a pure pass-through with no extra parsing logic of its own:
```go
func LegacyAppearanceToSpecForMigration(a LegacyAppearanceSpecForMigration) AppearanceSpec {
	return legacyAppearanceToSpec(a)
}
```
and in the script, parse `a.Albedo`/`a.Emissive` hex strings to `types.Rgba` (import
`oblikovati.org/api/types`, use `types.ParseHex`) before constructing
`material.LegacyAppearanceSpecForMigration{...}` — mirrors exactly what the OLD
`recipeToAppearance` used to do.

- [ ] **Step 2: Run the script over all 7 catalog files**

```bash
for f in model/material/catalog/*.yaml; do
  go run ./model/material/cmd/convertcatalog "$f"
done
```

- [ ] **Step 3: Hand-review a representative sample**

Read `model/material/catalog/01-metals.yaml` in full after conversion. Confirm:
comments at the top of the file (property sources, appearance notes) are still
present (the script only rewrites the `appearances:` list, not surrounding YAML —
if `yamlcodec.Marshal` strips comments, note this as a real loss and decide whether
to manually re-add the file header comment block after the script runs, since these
files' comments document real material-property sourcing that shouldn't silently
disappear). Confirm each entry's `base.color` matches the old `albedo` (gamma-decoded
to linear — spot check one value: aluminum's `#f4f5f5ff` should decode to
approximately `{R: 0.887, G: 0.892, B: 0.892}` via `pow(v/255, 2.2)`), `base.metalness
== 1.0` for every pure metal, `specular.roughness` matches the old `roughness`,
every other group (`transmission`/`subsurface`/`coat`/`fuzz`/`thinFilm`) is at its
spec default (all weights 0).

- [ ] **Step 4: Re-add stripped file-header comments if needed**

If Step 3 found the YAML marshaler stripped the top-of-file comment blocks, manually
re-insert them at the top of each of the 7 files (copy from `git show
HEAD~N:model/material/catalog/01-metals.yaml` etc., for each file, before the script
ran — use `git diff` against the pre-script commit to recover exact original text).

- [ ] **Step 5: Delete the script and the shim**

```bash
rm -rf model/material/cmd
```
Remove the `LegacyAppearanceSpecForMigration`/`LegacyAppearanceToSpecForMigration`/
`AppearanceRecipeForMigration` shim from `legacy_migrate.go`.

- [ ] **Step 6: Rewrite `catalog_test.go`**

Update any test asserting the old 5-scalar catalog shape to assert against the new
9-group one (e.g. a test checking "every catalog appearance has a valid albedo hex"
becomes "every catalog appearance's `base.color` channels are in `[0,1]`"). Add
`TestCatalogAppearancesLoadWithoutError` if no equivalent exists — load the real
embedded catalog via `NewLibrary()`, assert `len(lib.Appearances()) > 0` and every
entry's `Source() == SourceBuiltin`.

- [ ] **Step 7: Build, test, lint**

```bash
go build ./... 2>&1
go test ./model/material/... 2>&1
golangci-lint run ./model/material/... 2>&1
```
Expected: all clean. This is the first point in the plan where `model/material`
should be fully green again.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
Migrate the 7 built-in catalog files to the native OpenPBR schema (M46-F03)

Rewrites every appearances: entry from the old 5-scalar shape to the
full 9-group AppearanceRecipe shape, via a throwaway conversion script
(deleted in this same commit) reusing legacyAppearanceToSpec's exact
mapping. Hand-reviewed against the pre-migration values.
EOF
)"
```

---

## F04 — Persistence: one-time migration for old `.obk`/project-library data

### Task 10: Old-format fixtures

**Files:**
- Create: `test-utilities/openpbr-appearance-migration/old-document.obk`.
- Create: `test-utilities/openpbr-appearance-migration/old-project-library.yaml`.

**Interfaces:**
- Consumes: nothing (raw fixture data).
- Produces: two real, hand-authored files in the OLD pre-M46 shape, checked in for
  Task 11's migration test to load.

- [ ] **Step 1: Author `old-document.obk`**

Write a minimal but real `.obk` document whose materials section uses the
PRE-CONSOLIDATION shape: a `recipe.materials.appearances` entry with the OLD
5-scalar keys (`id`, `name`, `albedo`, `metallic`, `roughness`, `opacity`) and an
`assignments.partAppearance` referencing that custom id. Base the document
structure on an existing minimal `.obk` fixture already in the repo — find one via
`find test-utilities -iname "*.obk" | head -3` and copy its non-materials structure
(origin/parameters/solidBodies/sketch/extrusion), replacing only the materials
section.

- [ ] **Step 2: Author `old-project-library.yaml`**

Write a minimal project-library YAML (same shape `model/material/store.go` reads —
check that file's `Load`/`Save` for the exact top-level structure) with one custom
appearance in the OLD 5-scalar shape and no OpenPBR-shaped entries.

- [ ] **Step 3: Commit**

```bash
git add test-utilities/openpbr-appearance-migration/
git commit -m "test-utilities: old-format appearance fixtures for the M46-F04 migration test"
```

### Task 11: Shape-sniffing migration in `recipe.go`/`store.go`

**Files:**
- Modify: `model/material/recipe.go` — `ApplyRecipe`'s appearance-loading loop.
- Modify: `model/material/store.go` — the project-library load path (check its
  current function name via `grep -n "func.*Load" model/material/store.go` before
  writing this task's exact diff — the store's load function needs the same
  shape-sniffing treatment `ApplyRecipe` gets).
- Modify: `model/material/recipe_test.go`, `model/material/store_test.go`.

**Interfaces:**
- Consumes: the fixtures from Task 10, `legacyAppearanceToSpec` from Task 8.
- Produces: `ApplyRecipe`/the project-library loader transparently upgrade
  old-shaped embedded appearance YAML on read; no wire/contract change (this is
  entirely internal to how a `RecipeData`/project-library YAML blob gets parsed).

- [ ] **Step 1: Design the shape-sniff**

Since `yamlcodec.Unmarshal` into the NEW `AppearanceRecipe` struct would silently
leave every group at its zero value for an old-shaped entry (no error — YAML is
permissive about unknown/missing keys), detect old-shaped entries by unmarshaling
each appearance list item into a `map[string]any` FIRST, checking for the presence
of a top-level `albedo` key (old shape) vs `base` key (new shape):
```go
// appearanceRecipeShape reports which of the two on-disk appearance shapes a raw
// YAML map represents — "legacy" (pre-M46, 5-scalar, has a top-level "albedo" key)
// or "native" (9-group, has a top-level "base" key). Used only by the one-time
// migration path (F04); a normally-saved document/project-library file is always
// "native".
func appearanceRecipeShape(raw map[string]any) string {
	if _, ok := raw["albedo"]; ok {
		return "legacy"
	}
	return "native"
}
```

- [ ] **Step 2: Write the failing test**

In `recipe_test.go`:
```go
func TestApplyRecipeMigratesLegacyShapedAppearance(t *testing.T) {
	raw := []byte(`
appearances:
  - {id: my-custom-red, name: My Custom Red, albedo: "#c02020ff", metallic: 0.0, roughness: 0.4, opacity: 1.0}
assignments:
  partAppearance: my-custom-red
`)
	var data RecipeData
	if err := yamlcodec.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	set, assign := NewAssetSet(), NewAssignmentStore()
	if err := ApplyRecipe(data, set, assign); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	a, ok := set.Appearance("my-custom-red")
	if !ok {
		t.Fatal("legacy-shaped appearance was not migrated into the asset set")
	}
	if a.Base().Metalness != 0 || a.Base().Color == (Color3{}) {
		t.Errorf("migrated appearance base = %+v, want a non-zero color and metalness 0", a.Base())
	}
	if got := assign.PartAppearance(); got != "my-custom-red" {
		t.Errorf("PartAppearance() = %q, want %q (assignment ids don't change shape)", got, "my-custom-red")
	}
}
```
(`yamlcodec` must be imported in the test file — check if already imported, add if
not.)

- [ ] **Step 3: Run — expect it to fail differently than expected**

```bash
go test ./model/material/... -run TestApplyRecipeMigratesLegacyShapedAppearance -v 2>&1
```
Expected: the raw YAML unmarshals into `RecipeData.Appearances[0]` as an
`AppearanceRecipe` with every group zero-valued (silently WRONG, not a load error) —
`set.Appearance("my-custom-red")` succeeds but with a black/default appearance, so
the `a.Base().Color == (Color3{})` check fails, proving the bug this task fixes.

- [ ] **Step 4: Implement the migration in `ApplyRecipe`**

`RecipeData.Appearances []AppearanceRecipe` already lost the raw-map information by
the time `ApplyRecipe` runs (it's already been typed-unmarshaled). Move the
shape-sniff EARLIER, into a new function that unmarshals the raw YAML bytes twice —
once loosely, once typed — OR (simpler, avoids a second unmarshal pass) change
`RecipeData.Appearances`'s YAML unmarshaling to go through a custom
`UnmarshalYAML` that shape-sniffs per-entry. Prefer the custom-unmarshal approach
since it keeps `ApplyRecipe` itself unchanged (the migration happens transparently
at parse time, not at apply time):

Add to `recipe.go`:
```go
// UnmarshalYAML shape-sniffs each entry (M46-F04): a pre-consolidation document or
// project-library file has appearance entries in the OLD 5-scalar shape (a
// top-level "albedo" key), which this transparently upgrades via
// legacyAppearanceToSpec so an old file loads correctly instead of silently
// producing all-zero-value appearances.
func (r *AppearanceRecipe) UnmarshalYAML(unmarshal func(any) error) error {
	var raw map[string]any
	if err := unmarshal(&raw); err != nil {
		return err
	}
	if appearanceRecipeShape(raw) == "native" {
		type plain AppearanceRecipe // avoid recursing into this UnmarshalYAML
		var p plain
		if err := unmarshal(&p); err != nil {
			return err
		}
		*r = AppearanceRecipe(p)
		return nil
	}
	var legacy struct {
		ID          string  `yaml:"id"`
		DisplayName string  `yaml:"name"`
		Albedo      string  `yaml:"albedo"`
		Metallic    float32 `yaml:"metallic"`
		Roughness   float32 `yaml:"roughness"`
		Emissive    string  `yaml:"emissive,omitempty"`
		Opacity     float32 `yaml:"opacity"`
	}
	if err := unmarshal(&legacy); err != nil {
		return err
	}
	albedo, err := types.ParseHex(legacy.Albedo)
	if err != nil {
		return fmt.Errorf("material: legacy appearance %q: albedo: %w", legacy.ID, err)
	}
	emissive := types.Rgba{A: 1}
	if legacy.Emissive != "" {
		if emissive, err = types.ParseHex(legacy.Emissive); err != nil {
			return fmt.Errorf("material: legacy appearance %q: emissive: %w", legacy.ID, err)
		}
	}
	spec := legacyAppearanceToSpec(legacyAppearanceSpec{
		DisplayName: legacy.DisplayName, Albedo: albedo, Metallic: legacy.Metallic,
		Roughness: legacy.Roughness, Emissive: emissive, Opacity: legacy.Opacity,
	})
	*r = appearanceToRecipe(NewAppearance(legacy.ID, SourceBuiltin, spec))
	r.ID = legacy.ID
	return nil
}
```
Check `yamlcodec`'s actual unmarshal-hook signature before finalizing this (it may
wrap `gopkg.in/yaml.v3`, whose custom-unmarshaler interface is
`UnmarshalYAML(value *yaml.Node) error`, not the older
`UnmarshalYAML(unmarshal func(any) error) error` shown above — read
`yamlcodec`'s own source, e.g. `grep -rn "UnmarshalYAML\|yaml.Node" yamlcodec/*.go`,
and adapt this step's code to whichever hook signature it actually exposes; the
shape-sniff LOGIC (check for `albedo` vs `base`) stays the same regardless of which
signature is used, only the low-level (un)marshal calls differ. Add `"oblikovati.org/api/types"`
and `"fmt"` imports to `recipe.go` if not already present.

- [ ] **Step 5: Run the test again**

```bash
go test ./model/material/... -run TestApplyRecipeMigratesLegacyShapedAppearance -v 2>&1
```
Expected: PASS.

- [ ] **Step 6: Apply the same fix to `store.go`'s project-library loader**

Read `model/material/store.go`'s load function. If it parses appearances through the
SAME `AppearanceRecipe` type (likely, since `openpbr_store.go`'s
`projectOpenPBRAppearances` already showed a shared pattern), the `UnmarshalYAML`
hook from Step 4 covers it for free — no separate change needed. If it uses a
DIFFERENT/separate recipe type for project-library storage, apply the identical
shape-sniff there. Verify with a test using `test-utilities/openpbr-appearance-migration/old-project-library.yaml`
from Task 10:
```go
func TestLoadProjectLibraryMigratesLegacyShapedAppearance(t *testing.T) {
	lib := NewLibrary()
	store := &Store{Path: "../../test-utilities/openpbr-appearance-migration/old-project-library.yaml"}
	if err := store.Load(lib); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// assert the custom appearance from the fixture resolved to a non-zero-value
	// native Appearance, same shape as the recipe.go test above
}
```
(Check `Store`'s actual field/method names via `grep -n "type Store\|func.*Store.*Load" model/material/store.go`
before finalizing this test — adapt to the real API.)

- [ ] **Step 7: Full package test + lint**

```bash
go test ./model/material/... 2>&1
golangci-lint run ./model/material/... 2>&1
```

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
One-time migration for old-shaped embedded/project appearances (M46-F04)

AppearanceRecipe.UnmarshalYAML shape-sniffs each entry (a top-level
"albedo" key marks the pre-consolidation 5-scalar shape) and
transparently upgrades it through legacyAppearanceToSpec, so an old
.obk or project-library file loads correctly instead of silently
producing all-zero-value appearances. Assignment-id fields don't
change shape, so they need no migration.
EOF
)"
```

---

## F05 — Router + UI: delete legacy, promote OpenPBR to sole tab

### Task 12: `addin/router` — merge and rename

**Files:**
- Modify: `addin/router/material.go` — delete the appearance-specific handlers, keep
  the material-specific ones.
- Delete: `addin/router/openpbr_material.go` (its content moves into `material.go`,
  renamed).
- Modify: `addin/router/router.go` — update the registration block.
- Modify: `addin/router/router_test.go`, delete
  `addin/router/openpbr_material_test.go` (merge its cases into whatever test file
  covers `material.go`'s handlers — check
  `grep -rln "func TestAssignAppearance\|func TestListAppearances" addin/router/*_test.go`
  to find it, likely `router_test.go` or a dedicated `material_test.go` that may not
  exist yet).

**Interfaces:**
- Consumes: `wire.AppearanceInfo`/`MethodAppearances*`/`MethodModelAssignAppearance`
  from F01, `s.Materials().Appearances()`/`s.DuplicateAppearance()`/
  `s.UpdateAppearance()`/`s.AssignAppearance()` from Task 14.
- Produces: `appearanceInfo(*material.Appearance) wire.AppearanceInfo`,
  `listAppearances`, `getAppearance`, `createAppearance`, `updateAppearance`,
  `assignAppearance` — same function names as today, now operating on the 9-group
  type and renamed wire DTOs.

- [ ] **Step 1: Delete `material.go`'s legacy appearance handlers**

In `material.go`, delete `appearanceInfo`, `listAppearances`, `getAppearance`,
`createAppearance`, `updateAppearance`, `assignAppearance` (the 5-scalar-shaped
ones). Keep `materialInfo`, `listMaterials`, `getMaterial`, `createMaterial`,
`updateMaterial`, `assignMaterial`, `physicalProperties` untouched.

- [ ] **Step 2: Move and rename `openpbr_material.go`'s content into `material.go`**

Append `openpbr_material.go`'s functions into `material.go`, renaming:
`openPBRAppearanceInfo`→`appearanceInfo`, `listOpenPBRAppearances`→
`listAppearances`, `getOpenPBRAppearance`→`getAppearance`,
`createOpenPBRAppearance`→`createAppearance`, `updateOpenPBRAppearance`→
`updateAppearance`, `assignOpenPBRAppearance`→`assignAppearance`. Update their
bodies' type/wire references (`material.OpenPBRAppearance`→`material.Appearance`,
`wire.OpenPBRAppearanceInfo`→`wire.AppearanceInfo`,
`wire.ListOpenPBRAppearancesResult`→`wire.ListAppearancesResult`,
`wire.CreateOpenPBRAppearanceArgs`→`wire.CreateAppearanceArgs`,
`wire.UpdateOpenPBRAppearanceArgs`→`wire.UpdateAppearanceArgs`,
`wire.AssignOpenPBRAppearanceArgs`→`wire.AssignAppearanceArgs`,
`s.Materials().OpenPBRAppearance(...)`/`OpenPBRAppearances()`→
`s.Materials().Appearance(...)`/`Appearances()`, `s.DuplicateOpenPBRAppearance`→
`s.DuplicateAppearance`, `material.OpenPBRAppearanceSpec`→`material.AppearanceSpec`,
`s.UpdateOpenPBRAppearance`→`s.UpdateAppearance`, `s.AssignOpenPBRAppearance`→
`s.AssignAppearance`. Update error message strings from
`"openpbrAppearances.get: ..."`/`"openpbrAppearances.update: ..."` to
`"appearances.get: ..."`/`"appearances.update: ..."`. Delete the file's now-stale
doc comment ("Mirrors appearanceInfo for the full OpenPBR lobe set" — there's
nothing left to mirror, this IS the appearance handler now).

```bash
rm addin/router/openpbr_material.go
```

- [ ] **Step 3: Update `router.go`'s registration**

Replace lines currently at `addin/router/router.go:451-454` and `:457-460`:
```go
	r.readOnly(wire.MethodAppearancesList, listAppearances)
	r.readOnly(wire.MethodAppearancesGet, typed(getAppearance))
	r.readOnly(wire.MethodAppearancesCreate, typed(createAppearance))
	r.readOnly(wire.MethodAppearancesUpdate, typed(updateAppearance))
```
(keep this block AS-IS — the method constants Task 2 renamed already point at the
right strings, and the handler function names are unchanged after Step 2's rename;
DELETE the old `MethodOpenPBRAppearances*` registration block that followed it).
Replace `router.go:466-467`:
```go
	r.mutating(wire.MethodModelAssignAppearance, "Assign Appearance", typed(assignAppearance))
	r.mutating(wire.MethodModelAssignOpenPBRAppearance, "Assign OpenPBR Appearance", typed(assignOpenPBRAppearance))
```
with:
```go
	r.mutating(wire.MethodModelAssignAppearance, "Assign Appearance", typed(assignAppearance))
```

- [ ] **Step 4: Build**

```bash
go build ./addin/... 2>&1
```

- [ ] **Step 5: Merge/update tests**

```bash
grep -rln "TestAssignAppearance\|TestListAppearances\|TestGetAppearance\|TestCreateAppearance\|TestUpdateAppearance" addin/router/*_test.go
```
For whichever file(s) that finds: delete tests exercising the OLD 5-scalar handlers,
keep/rename tests from `addin/router/openpbr_material_test.go` (merge its content
into that file, renaming `TestModelAssignOpenPBRAppearance`→
`TestModelAssignAppearance` etc., updating type/wire references per Step 2's
pattern) then:
```bash
rm addin/router/openpbr_material_test.go
```
Check `addin/router/api_parity_test.go`'s `notYetHandled` map doesn't need an entry
(it's currently empty — should stay empty, since every renamed method still has a
handler registered).

- [ ] **Step 6: Run and commit**

```bash
go test ./addin/... 2>&1
golangci-lint run ./addin/... 2>&1
git add -A
git commit -m "addin/router: merge appearance handlers into material.go, rename (M46-F05)"
```

### Task 13: `head/ui` — delete legacy editor, promote OpenPBR to sole tab

**Files:**
- Delete: `head/ui/appearance_editor.go`.
- Modify: `head/ui/openpbr_appearance_editor.go` → rename to
  `head/ui/appearance_editor.go` (reclaiming the name).
- Modify: `head/ui/materials_window.go` — drop the tab switcher's "Appearances"
  entry, rename the "OpenPBR" tab label.
- Delete: `head/ui/openpbr_appearance_editor_test.go`'s now-superseded cases (merge
  into whatever test file covers the promoted editor — likely renamed alongside the
  source file).
- Delete `head/ui/appearance_editor_test.go` if it exists (check first) — same
  merge/delete treatment as the router task.

**Interfaces:**
- Consumes: `s.Materials()`, `s.DuplicateAppearance()`, `s.UpdateAppearance()`,
  `s.AssignAppearance()` from Task 14 (renamed session methods).
- Produces: `drawAppearanceTabContent(s appearanceEditorSession)` (renamed from
  `drawOpenPBRTabContent`), a single `appearanceEditorSession` interface (renamed
  from `openPBREditorSession`), `selectedAppearance string` (single package-level
  var, reclaiming the name from the deleted legacy tab's own `selectedAppearance`).

- [ ] **Step 1: Delete the legacy editor**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati
rm head/ui/appearance_editor.go
```

- [ ] **Step 2: Promote and rename the OpenPBR editor**

```bash
git mv head/ui/openpbr_appearance_editor.go head/ui/appearance_editor.go
```
In the moved file: rename `var selectedOpenPBR string`→`var selectedAppearance
string`; rename `type openPBREditorSession interface`→`type
appearanceEditorSession interface`, and its methods:
```go
type appearanceEditorSession interface {
	Materials() *material.Library
	DuplicateAppearance(baseID, name string) (*material.Appearance, error)
	UpdateAppearance(id string, spec material.AppearanceSpec)
	AssignAppearance(scope, key, appearanceID string) error
}
```
Rename `drawOpenPBRTabContent`→`drawAppearanceTabContent`,
`drawOpenPBRSelector`→`drawAppearanceSelector`, and every other `OpenPBR`-prefixed
function in the file, dropping the prefix. Update every body reference from
`lib.OpenPBRAppearance(...)`/`OpenPBRAppearances()` to `lib.Appearance(...)`/
`Appearances()`, `s.DuplicateOpenPBRAppearance`→`s.DuplicateAppearance`,
`s.UpdateOpenPBRAppearance`→`s.UpdateAppearance`, `s.AssignOpenPBRAppearance`→
`s.AssignAppearance`, `material.OpenPBRAppearanceSpec`→`material.AppearanceSpec`.
Update the file's doc comment (drop "a full-spec sibling of appearance_editor.go's
metallic-roughness panel" — there's no sibling anymore, this IS the appearance
editor).

- [ ] **Step 3: Update `materials_window.go`**

Delete the `selectedAppearance`/`apprNameBuf` package-level vars declared in THIS
file (they belonged to the now-deleted legacy tab; the renamed
`selectedAppearance` from Step 2 lives in `appearance_editor.go` instead — having
both declared would be a duplicate-symbol compile error, which is exactly the
signal that confirms Step 2 must land before this step). Replace `drawMaterialsBody`'s
tab block:
```go
	if native.BeginTabItem("Materials") {
		drawMaterialTab(s)
		native.EndTabItem()
	}
	if native.BeginTabItem("Appearances") {
		drawAppearanceTabContent(s)
		native.EndTabItem()
	}
	if native.BeginTabItem("Physical") {
		drawPhysicalReadout(s)
		native.EndTabItem()
	}
```
(single "Appearances" tab calling the renamed `drawAppearanceTabContent`, "OpenPBR"
tab entry deleted). `syncMaterialSelection`'s `s.ActivePartAppearanceID()` call at
line 46 is unchanged (that session method's name never had an "OpenPBR" prefix to
begin with).

- [ ] **Step 4: Check the session-coupling ratchet**

This file swap changes how many `*app.Session` occurrences exist in `head/ui/*.go`
(deleting `appearance_editor.go`'s OLD content removes some, the promoted file adds
back roughly the same shape it already had as `openpbr_appearance_editor.go`, which
was ALREADY accounted for in the ratchet's pinned constant this session). Run:
```bash
cd head && go test ./archguard/... 2>&1
```
If it fails (ratchet count changed), follow this repo's established narrow-interface
pattern (already used by `appearanceEditorSession` itself) to bring it back to the
pinned value — do NOT raise the pin.

- [ ] **Step 5: Build**

```bash
go build ./... 2>&1
```

- [ ] **Step 6: Merge/update tests**

```bash
grep -n "func Test" head/ui/openpbr_appearance_editor_test.go head/ui/appearance_editor_test.go 2>/dev/null
```
Merge whichever legacy-editor-only tests still make sense (e.g. "every group
renders" smoke test) into the promoted file's test (rename
`head/ui/openpbr_appearance_editor_test.go`→`head/ui/appearance_editor_test.go`,
deleting the old file at that path first if one exists), updating symbol references
per Step 2's rename pattern. Delete `head/ui/materials_window_test.go` cases that
specifically exercised the now-deleted "Appearances" legacy tab, keeping cases for
"Materials"/"Physical" tabs and the single "Appearances" (renamed) tab.

- [ ] **Step 7: Run and commit**

```bash
cd head && go test ./ui/... 2>&1
golangci-lint run ./ui/... 2>&1
git add -A
git commit -m "head/ui: delete legacy appearance editor, promote OpenPBR editor to sole tab (M46-F05)"
```

### Task 14: `app` session methods — merge and rename

**Files:**
- Modify: `app/material_ops.go` — delete legacy `AssignAppearance`/
  `embedAppearance`, rename the OpenPBR ones.
- Modify: `app/materials.go` — delete legacy `DuplicateAppearance`/`UpdateAppearance`,
  `appearanceSurface`, `openPBRAppearanceSurface`; rename.
- Delete: `app/openpbr_materials.go` (merged into `materials.go`).
- Modify: `app/materials_test.go`, `app/openpbr_materials_test.go` (merge, delete the
  latter), `app/material_active_test.go`, `app/appearance_render_probe_test.go`,
  `app/assembly_render_test.go`.

**Interfaces:**
- Consumes: `material.Appearance`/`AppearanceSpec`/`AssetLookup` from F02.
- Produces: `Session.DuplicateAppearance(baseID, name string) (*material.Appearance,
  error)`, `Session.UpdateAppearance(id string, spec material.AppearanceSpec)`,
  `Session.AssignAppearance(scope, key, appearanceID string) error`,
  `Session.ActivePartAppearanceID() string` (unchanged name/signature),
  `Session.SurfaceLookup()`/`partSurfaceLookup` simplified to a SINGLE resolution
  path (the `if opbr, ok := assign.EffectiveOpenPBRAppearance(...); ok {...}
  fallback to legacy` two-step from #2150's fix collapses back to one call, since
  there's only one chain now).

- [ ] **Step 1: `app/materials.go`**

Delete `DuplicateAppearance`, `UpdateAppearance` (the legacy 5-scalar-typed ones —
their bodies call `s.Materials().DuplicateAppearance`/`EditAppearance`, which after
F02 already operate on the 9-group type, so actually KEEP these two functions
UNCHANGED in name/signature — they already say `*material.Appearance`/
`material.AppearanceSpec` as their parameter/return types via Go's type identity,
since F02 renamed the underlying `material` package types out from under them; no
source edit needed here beyond confirming they compile). Delete `appearanceSurface`
(the legacy sRGB-passthrough one) and `openPBRAppearanceSurface` (rename it to
`appearanceSurface`, since it's now the only conversion function):
```go
func appearanceSurface(a *material.Appearance) renderer.Surface {
	base, spec, geo := a.Base(), a.Specular(), a.Geometry()
	em := a.Emission()
	emissive := material.Color3{R: em.Color.R * em.Luminance, G: em.Color.G * em.Luminance, B: em.Color.B * em.Luminance}
	return renderer.Surface{
		Albedo:    encodeSRGBColor(base.Color),
		Metallic:  base.Metalness,
		Roughness: spec.Roughness,
		Emissive:  encodeSRGB3(emissive),
		Opacity:   geo.Opacity,
	}
}
```
(`encodeSRGBChannel`/`encodeSRGB3`/`encodeSRGBColor` helpers stay unchanged — they
have no dependency on the deleted type).

Simplify `partSurfaceLookup`'s two-step resolution back to one call:
```go
func (s *Session) partSurfaceLookup(part *compdef.PartComponentDefinition) renderer.SurfaceLookup {
	look := material.MergedLookup{Embedded: part.Assets(), Catalog: s.Materials()}
	assign := part.Assignments()
	return func(b *topo.Body) renderer.Surface {
		if name, ok := s.BodyColorStyle(string(b.ReferenceKey())); ok {
			if cs, found := s.styles.ByName(name); found {
				return styleSurface(cs)
			}
		}
		key := material.RefKey(b.ReferenceKey())
		appr := assign.EffectiveAppearance(look, key, "")
		return appearanceSurface(appr)
	}
}
```
(drops the `if opbr, ok := assign.EffectiveOpenPBRAppearance(...); ok {...}` branch
entirely — `EffectiveAppearance` from Task 5 already resolves the single chain).
`assemblySurfaceLookup`'s `fallback := appearanceSurface(material.MergedLookup{Catalog:
s.Materials()}.DefaultAppearance())` line is unchanged (already calls the
now-renamed function).

- [ ] **Step 2: `app/material_ops.go`**

Delete the legacy `AssignAppearance`/`embedAppearance` (their bodies already say
`material.Appearance`/`s.Materials().Appearance`, which after F02 IS the 9-group
type — so, like Step 1, confirm these compile unchanged rather than deleting them;
what actually needs deleting is `AssignOpenPBRAppearance`/`embedOpenPBRAppearance`,
now fully redundant with `AssignAppearance`/`embedAppearance`):
```bash
grep -n "func.*AssignOpenPBRAppearance\|func.*embedOpenPBRAppearance" app/material_ops.go
```
Delete both functions. `embedMaterial`'s call to `s.embedAppearance(part,
m.AppearanceID())` is unchanged (already calls the surviving function).

- [ ] **Step 3: Delete `app/openpbr_materials.go`**

Its two functions (`DuplicateOpenPBRAppearance`, `UpdateOpenPBRAppearance`) are now
fully redundant with `materials.go`'s (post-Step-1) `DuplicateAppearance`/
`UpdateAppearance`:
```bash
rm app/openpbr_materials.go
```

- [ ] **Step 4: Build**

```bash
go build ./... 2>&1
```

- [ ] **Step 5: Merge/update tests**

Delete `app/openpbr_materials_test.go`'s tests that are now pure duplicates of
`app/materials_test.go`'s (post-#2150,
`TestSurfaceLookupResolvesAssignedOpenPBRAppearance`-shaped tests collapse into
whatever `TestSurfaceLookupResolvesAssignedAppearance` already covers — check both
files for overlap before deleting either). Rename any surviving
`Test*OpenPBRAppearance*`→`Test*Appearance*`, delete the file if everything in it
became redundant:
```bash
rm app/openpbr_materials_test.go   # only if Step 5's review found full redundancy
```
Update `app/material_active_test.go`, `app/appearance_render_probe_test.go`,
`app/assembly_render_test.go` for any `OpenPBRAppearance`-named symbol references.

- [ ] **Step 6: Run and commit**

```bash
go test ./app/... 2>&1
golangci-lint run ./app/... 2>&1
git add -A
git commit -m "$(cat <<'EOF'
app: merge appearance session methods, simplify SurfaceLookup (M46-F05)

partSurfaceLookup drops the two-step "try OpenPBR chain, fall back to
legacy" resolution from #2150's fix — EffectiveAppearance is once
again the only resolver, since there's only one chain now.
EOF
)"
```

### Task 15: Remove the now-unnecessary lint exclusions

**Files:**
- Modify: `.golangci.yml`.

**Interfaces:** none.

- [ ] **Step 1: Remove the deprecated-Appearance exclusion rule**

The rule added earlier this session (`text: "SA1019:.*Appearance is deprecated"`,
`path: (app/materials\.go|model/material/appearance\.go|addin/router/material\.go)`)
is dead — `material.Appearance`/`contract.Appearance` are no longer deprecated (the
OLD deprecated type is deleted; the current `Appearance` is the promoted OpenPBR
one, never marked deprecated). Delete this rule block from `.golangci.yml` entirely.

- [ ] **Step 2: Full lint**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati
golangci-lint run ./... 2>&1
cd head && golangci-lint run ./... 2>&1
```
Expected: 0 issues in both. If any SA1019 warnings resurface, they indicate a
lingering reference somewhere Tasks 5-14 missed — grep for `OpenPBRAppearance\b`
(excluding the 9 group type names) across the whole repo and fix.

- [ ] **Step 3: Commit**

```bash
git add .golangci.yml
git commit -m "Remove the now-dead deprecated-Appearance lint exclusion (M46-F05)"
```

---

## F06 — Live verification + cleanup

### Task 16: Full-repo build/vet/test/lint sweep

**Files:** none (verification task).

- [ ] **Step 1: Root module**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati
go build ./... && go vet ./... && go test ./... 2>&1
golangci-lint run ./... 2>&1
```

- [ ] **Step 2: `head` module**

```bash
cd head
go build ./... 2>&1
go test ./... 2>&1
golangci-lint run ./... 2>&1
cd .. && go test ./archguard/... 2>&1
```

- [ ] **Step 3: Grep sweep for leftover old names**

```bash
grep -rn "OpenPBRAppearance\b\|OpenPBRAppearances\b\|openpbrAppearances\." --include="*.go" . | grep -v "types\.OpenPBR[A-Z]\|OpenPBRAppearanceInfo\b"
```
Expected: empty (any hit is a missed rename from Tasks 5-15).

- [ ] **Step 4: Fix any fallout, re-run until clean**

Repeat Steps 1-3 until all green.

### Task 17: Live MCP verification

**Files:** none (uses the same throwaway MCP-bridge driving technique as M45
PBI-354 — see that PBI's own work for the exact `go.work` repoint + `mcplive`-style
driver pattern; this task doesn't re-derive it from scratch).

- [ ] **Step 1: Build and install the MCP bridge against this branch**

Follow the same procedure M45 PBI-354 used: temporarily repoint
`Oblikovati.AddIns.MCPBridge`'s `go.work` replaces at this worktree's `Oblikovati`/
`Oblikovati.API` paths, `go generate ./...` to regenerate `tools_generated.go`
against the renamed `assign_appearance`/`list_appearances`/etc. MCP tool names,
`make install` into this worktree's `head/addins`.

- [ ] **Step 2: Launch and drive**

Launch `oblikovati-head` on a free `OBK_MCP_ADDR` port (check `ss -ltnp | grep 7800`
first — another session may hold the default port). Build a simple part, assign a
distinctly-colored `Appearance` (via the renamed `assign_appearance` tool) to it,
switch to Realistic mode, capture via `capture_window` (not `capture_viewport` —
#2149's gap is unrelated to this work and still open). Confirm the assigned color
renders correctly (same visual proof M45 PBI-354 used) and the head process log has
no Vulkan validation errors.

- [ ] **Step 3: Clean up**

Kill the live `oblikovati-head` instance, restore `Oblikovati.AddIns.MCPBridge`'s
`go.work` to its original committed-remote-path state, delete any throwaway driver
files created for this step.

### Task 18: Open the API PR (done in F01/Task 3) and push the GPL commits

**Files:** none (process task — F01's PR already opened/merged in Task 3; this task
is GPL-side only).

- [ ] **Step 1: Push the accumulated GPL commits**

```bash
cd /home/vmiguel/git/oblikovati-workspace/.worktrees/m45-openpbr/Oblikovati
git push origin feat/m45-openpbr-host
```

- [ ] **Step 2: Update PR#2152's description**

```bash
gh pr view 2152 --json body -q .body > /tmp/pr2152_body.md
```
Append a section describing the M46 consolidation work (delete legacy Appearance,
rename OpenPBRAppearance to Appearance, catalog migration, old-file migration),
referencing the spec doc, then:
```bash
gh pr edit 2152 --body-file /tmp/pr2152_body.md
```

- [ ] **Step 3: Watch CI, merge when green**

```bash
gh pr checks 2152 --watch
gh pr merge 2152 --merge --delete-branch=false
```
