<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Native themes and colour schemes

Tracks Oblikovati #3805. This feature is in progress.

## Ownership

The application owns one theme library, independent of Part focus, document
transactions, and document save. GUI chrome consumes its semantic palette.
Viewport backgrounds and selection colours belong to the separate colour scheme
registry. A theme must not overwrite the selected scheme's background.

The reference is Go commit `aa8297d1b0ac33c024f19e54821536d76e8a447a`:
`theme/`, `head/ui/theme_apply.go`, and `model/colorscheme/`.

## Implemented boundaries

- Model: the exact Dark and Light XML assets, 39 API tokens, direct and derived
  colours, independent custom snapshots, library selection and revisions.
- XML: retained event documents preserve vendor namespaces, unknown attributes,
  comments and text. Save replaces root token snapshots. Input is bounded to
  16 MiB and 64 nested elements.
- GUI: the View picker emits an owner selection action; frame snapshots carry the
  resulting palette. 35 ImGui colour slots use the Go chrome bindings. The pinned ImGui version
  has older tab slot names and uses Text for its input cursor colour.
- Automation: canonical `theme.active` and `theme.list` DTOs and typed clients,
  with an explicit borrowed library on FileHost.
- Colour schemes: the separate owner registry contains all four Go palettes, with
  canonical list, get-active and set-active API methods. The View picker shares
  this registry with API clients. Martensite renders solid and vertical gradient
  backgrounds with sRGB conversion; scene submissions preserve the selection.
  Palette parity tests
  compare every channel, opacity and source through the public contract.
- Persistence: `preferences.yaml` uses Go's `activeTheme` field. Atomic selection
  saves retain unknown groups and publish memory only after replacement succeeds.
  Malformed or unreadable preferences return a named error before overwrite.

## Evidence

`crates/model/tests/theme/go_oracle.go` generates the checked-in Go baseline.
Tests compare every built-in token's raw channels and hex value, plus a sequence
of library selections, duplicates, custom edits and removals. XML tests verify
limits, first-child lookup, duplicate snapshot replacement and vendor retention.
GUI tests check unique slot ownership and exact palette copies. Typed API tests
check complete palettes, gallery order, custom edits and owner injection.
Preference tests inject read failures and each atomic write failure phase, then
check the stored bytes and in-memory selection.

## Remaining acceptance

Native startup loads the active preference and injects its library and selection
store into the owner; API and GUI share it. Script and add-in wrappers forward
selection to that owner. A failed save keeps the old live palette. The default
configuration root matches Go; `OBK_CONFIG_DIR` overrides it for isolated runs.
Startup also loads custom XML files in filename order. Valid files survive
malformed, missing and duplicate-file declines; their notices reach desktop
status. Custom snapshot export uses the atomic writer. The loaded-name-to-source map is
retained by storage so later edits update the original filename. OS XTest selection and
fresh-process reload of Workshop passed with a malformed neighbour present,
confirmed by MCPBridge queries and captured orange accents. An unrelated stored
preference group survived selection save.
The theme editor now exposes all 39 Go-labelled colour rows. Built-ins are
read-only. Custom edits preview through the owner, and Save writes the loaded
source file. Stale names, invalid colours and failed writes have named declines.
GUI tests cover modal fade completion, read-only controls, document closure and
theme-owner retirement. Live OS input verified preview without disk writes,
explicit Save, preserved vendor metadata and preferences, and a fresh-process
reopen of the edited colour. Both desktops closed normally.

Duplicate and Delete now call the same owner, with built-in deletion disabled.
New files receive independent source identities, so display-name slug collisions
cannot overwrite another theme. Delete uses the original source filename.
Creation and removal failures preserve the previous palette and source map.
A preference failure after a completed file change reports that partial outcome;
the gallery retains created themes and removes deleted ones. Unit tests cover
these boundaries and memory-only libraries. OS XTest created two similarly named
copies, reopened the selected copy, deleted it, and reopened Dark. The other copy
and original vendor XML remained intact. All three desktops closed normally.

Apply the selection/highlight colours to selection rendering. OS XTest selection of all four schemes has passed with Go
MCPBridge queries and pixel samples from completed window captures. High Contrast
stays black when the chrome theme changes; Presentation and Sky retain their
authored gradient directions. Native tests also cover gradient-to-solid changes,
scene refresh, no-op capture generations and a closed view.
The built-in selection path has passed OS XTest mouse input, Go MCPBridge theme
queries and window captures, atomic preference save, and a fresh-process Light
theme reopen. Both desktops closed normally. Run the full local gate, coverage and duplication
checks before pushing. Selection/highlight rendering and the full feature
acceptance remain incomplete.
