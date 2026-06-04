# keybinding-extraction

Extract keyboard shortcuts from major-CAD reference PDFs and map them onto
Oblikovati's canonical operations, so that a future preferences screen can offer
selectable **shortcut profiles** (Inventor / SolidWorks / NX / Solid Edge) and let
users edit their keybindings.

This is **data + tooling that feeds PBI-056/057** (ControlDefinitions + CommandManager
hotkey binding), parked next to those PBIs. It produces reviewable mapping data and the
generator for it, not engine code — the real keybinding model is the follow-up PBI work.

## Decisions baked in

- **Canonical operation vocabulary = Inventor's `CommandIDEnum`.** Every mapping
  target is an existing enum member (e.g. `kCreateExtrudeCommand`), parsed from
  `Oblikovati.Contracts/.../Enums/CommandIDEnum.cs` (183 members). We **never invent
  ids** — `build_profiles.py` aborts if `mappings/canonical.json` names an id that is
  not in the enum. Vendor actions with no enum counterpart are reported as unmapped,
  not forced.
- **Per-profile chord map.** A profile is `canonicalId -> {chord, alias}`, separate
  from the single typed `alias` the app's `CommandDefinition` carries today. `chord`
  is a key combo (`Ctrl+Shift+S`); `alias` is a typed sequence (Inventor `BA`,
  SolidWorks search alias `bom`).
- **Broaden to universal ops** where the enum allows it (Save/Open/Zoom/Pan/Rotate/
  Measure/Rebuild map cleanly). Undo/Redo/Copy/Paste/Cut/Delete, view orientations
  (Front/Top/Iso), Print, Help, selection-priority, Solid Edge synchronous-edit, and
  the Inventor mold/FEA/harness/simulation long tail have **no `CommandIDEnum`
  member** and therefore land in `generated/unmapped.md` for a later decision.

## Pipeline

```
python3 vocab.py > generated/vocab.json   # CommandIDEnum.cs  -> canonical vocabulary
python3 extract.py                  # NX/SolidEdge/SolidWorks PDFs -> generated/raw/*.json
python3 extract_inventor.py         # Inventor PDF      -> generated/raw/autodesk-inventor.json
python3 build_profiles.py           # raw + canonical.json -> profiles + reports
python3 chord.py                    # self-test the chord normalizer (doctest)
```

Source PDFs and `CommandIDEnum.cs` are read from the workspace root (resolved via
`Path.parents`, so the scripts work unchanged wherever this folder is checked out); only
`pdftotext` (poppler) and stdlib Python are required.

| File | Role |
|---|---|
| `vocab.py` | parse `CommandIDEnum.cs` into the canonical id vocabulary |
| `chord.py` | normalize vendor shortcut strings into one chord syntax (mods Ctrl→Alt→Shift→key); flags mouse/drag gestures |
| `extract.py` | parse the three clean tabular PDFs (NX, Solid Edge, SolidWorks) |
| `extract_inventor.py` | parse Inventor's two-column sticker sheet via its TAB+BEL key/name delimiter |
| `mappings/canonical.json` | **hand-authored** `canonicalId -> [vendor action spellings]` — the verifiable artifact |
| `build_profiles.py` | join + validate, emit `generated/profiles/*.json`, `generated/unmapped.md`, `generated/coverage.md` |

## Outputs (`generated/`)

- `profiles/<vendor>.json` — the per-vendor chord map: `{ canonicalId: {chord, alias, source} }`.
- `unmapped.md` — every vendor action with no canonical operation, grouped by vendor.
- `coverage.md` — mapped/unmapped counts and the canonical commands no vendor binds.
- `raw/<vendor>.json`, `vocab.json` — intermediates.

## Coverage (current)

| Vendor | Rows | Mapped | Notes |
|---|---|---|---|
| autodesk-inventor | 276 | 79 | richest; one-key + multi-char aliases |
| solidworks | 93 | 15 | many rows are UI toggles / orientations (no enum op) |
| siemens-nx | 73 | 7 | mostly menu/view/edit ops without enum ops |
| solidedge | 48 | 1 | almost entirely synchronous-edit modifiers & keypoint snaps |

88 of 183 canonical commands get a binding from at least one vendor.

## Known limitations (honest caveats for the implementation phase)

- **Flat profile vs. context-dependent keys.** Inventor reuses one-key shortcuts per
  environment (`X` = Fix-constraint in sketch *and* Trim; alias `CF` = Contour Flange
  *and* Create Fold). A flat `canonicalId -> chord` map keeps only the first; the real
  model must scope chords by environment. The app already has the `Environment`
  concept (`app/ribbon_key.go`) to hang this on.
- **Inventor extraction is best-effort.** The two-column art yields a few imperfect
  names (e.g. `HN` is really "Hole/Thread Notes" but reads as "Hole"; "Balloon, Bom").
  Categories are dropped for Inventor because column interleaving makes them
  unreliable — mapping keys on the action name instead.
- **`mappings/canonical.json` is curated, not exhaustive.** It maps only where a vendor
  action clearly denotes the same operation as an enum member. Widening it is a matter
  of adding synonyms; correctness is the priority over coverage.

## Next step (not done here)

Promote `generated/profiles/*.json` into a real keybinding feature: define the chord/profile
types in `Oblikovati.API/types`, load bundled vendor profiles, and add the preferences
plumbing to select/override them (ADR-0018 two-part split). See PBI-056/057
(ControlDefinitions + hotkey binding) which this feeds.
