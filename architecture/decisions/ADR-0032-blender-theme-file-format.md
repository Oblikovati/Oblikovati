# ADR-0032: Blender theme XML as the theme file format

- Status: accepted
- Date: 2026-06-11
- Issue: Oblikovati#655
- Supersedes: the YAML token→hex theme files of ADR-0021 (the semantic-token model
  itself stands)

## Context

ADR-0021 gave the shell a small frozen vocabulary of semantic color tokens
(`api/types.ThemeToken`) and stored themes as bespoke YAML files (`name` + token→hex
map), with the two built-ins hardcoded as Go maps in `theme/defaults.go`.

Blender has a mature, widely-adopted theme file standard: an XML document
(`<bpy><Theme>…`) of nested single-purpose elements whose attributes carry every color
of every editor. A large ecosystem of themes already exists in that format, and our
designers iterate on looks in Blender's theme editor. Recoloring our defaults
should be an asset edit, not a Go change.

## Decision

1. **The Blender theme XML document is the theme file format.** Custom themes are
   stored under `<config>/themes/*.xml`; the YAML theme format is dropped outright (no
   migration — alpha, no installed base). The `preferences.yaml` selected-theme
   pointer is unchanged.
2. **`theme/dark.xml` and `theme/light.xml` are the shipped built-ins**, embedded via
   `go:embed`, parsed once, and cloned per use. The file IS the theme: recoloring a
   default is an XML edit.
3. **The token vocabulary stays frozen.** A curated mapping (`theme/blendermap.go`)
   resolves each token from the document:
   - **Direct tokens** bind to exactly one Blender attribute (unique across the table,
     so an edit writes back to exactly one slot).
   - **Derived tokens** are colors Blender computes at draw time rather than stores
     (widget hover/press brightening, disabled dimming, grid emphasis); they derive
     from a direct token's resolved color (mix toward `chrome.text`, or alpha scale).
4. **Documents round-trip with full fidelity.** Files parse into a generic element
   tree (`theme/blenderxml`), never schema structs, so the hundreds of attributes we
   do not map survive load → edit → save. Editing a direct token writes through to its
   attribute, keeping the body a faithful Blender theme.
5. **Customs stay self-contained (ADR-0021) via one extension element.** Saving adds a
   single `<oblikovati_tokens name="…" chrome.text="#…" …/>` sibling of `<Theme>`
   holding the display name and a full token snapshot. On load: mapping first,
   snapshot on top. Derived-token edits persist only here. Stripping the element
   yields a stock Blender theme; a stock Blender export (no element) loads via the
   mapping alone and is named after its file base — any downloaded Blender theme can
   be dropped into the themes directory.
6. **Missing mapped slots fail loudly.** A document that lacks a mapped attribute is
   rejected at load with every missing path named, never rendered with fallback
   colors.

## Token mapping

Direct bindings (`ui` = `Theme/user_interface/ThemeUserInterface`, `v3d` =
`Theme/view_3d/ThemeView3D`, `wcol_*` under `ui`, attributes after `@`):

| Token | Blender source |
| --- | --- |
| chrome.window_bg | `ui@editor_border` (opaque) |
| chrome.panel_bg | `ui@panel_back` |
| chrome.header_bg | `ui@panel_header` |
| chrome.popup_bg | `wcol_menu_back@inner` |
| chrome.menu_bar_bg | `Theme/topbar/…/ThemeSpaceGeneric@header` |
| chrome.text | `wcol_regular@text` |
| chrome.border | `wcol_regular@outline` |
| chrome.control_bg | `wcol_text@inner` |
| chrome.button | `wcol_tool@inner` |
| chrome.button_active | `wcol_tool@inner_sel` (Blender's pressed state) |
| chrome.accent | `wcol_regular@inner_sel` (opaque) |
| chrome.scrollbar | `wcol_scroll@item` |
| viewport.bg | `v3d/space/…/gradients/…@high_gradient` (opaque) |
| viewport.grid_minor | `v3d@grid` |
| viewport.sketch_geometry | `v3d@wire_edit` (sketch ≙ edit-mode geometry) |
| viewport.sketch_selected | `v3d@edge_select` |
| viewport.sketch_candidate | `v3d@editmesh_active` |
| viewport.sketch_preview | `v3d@transform` |
| viewport.snap_glyph | `v3d@vertex_select` |
| viewport.dimension_driving | `v3d/space/ThemeSpaceGradient@text` |
| viewport.dimension_driven | `wcol_state@inner_driven` (Blender's "driven" state) |
| viewport.active_border | `ui@editor_outline_active` |
| gizmo.plane_faint | `ui@gizmo_primary` |
| gizmo.plane_hover | `ui@gizmo_hi` |
| gizmo.plane_selected | `v3d@object_active` |
| gizmo.plane_fill | `v3d@face` |
| gizmo.selection_highlight | `v3d@object_selected` |
| icon.tint | `wcol_toolbar_item@text` |

Derived bindings:

| Token | Derivation |
| --- | --- |
| chrome.text_disabled | chrome.text, alpha × 0.5 |
| chrome.control_hover | chrome.control_bg mixed 10% toward chrome.text |
| chrome.control_active | chrome.control_bg mixed 18% toward chrome.text |
| chrome.button_hover | chrome.button mixed 10% toward chrome.text |
| viewport.grid_major | viewport.grid_minor, alpha × 2 |
| viewport.grid_axis | viewport.grid_minor, alpha × 3 |
| icon.disabled | icon.tint, alpha × 0.5 |

Mixing toward the text color reproduces Blender's hover brightening in a
polarity-correct way: dark themes lighten, light themes darken.

## Consequences

- The shell's look is now authored in Blender's theme editor; the Go side only maps.
- The shipped themes inherit their source file's choices verbatim; visual fixes are
  asset edits. The dark file as exported had `#000000` gizmo borders and a near-zero
  `face` fill (work planes invisible on the dark canvas), tuned in place to the
  theme's fourth-axis gold (`gizmo_primary` `#edba18`, `gizmo_hi` `#ffd75e`, `face`
  `#edba1840`) — gold echoes the pre-migration translucent-orange planes and stays
  distinct from the purple selected state (`object_active`).
- `theme.New` remains as a palette-only seam for tests; such themes cannot be saved
  (no document) and the store rejects them loudly.
- Completeness is enforced twice: `TestEmbeddedThemesResolveCompletely` (every direct
  binding resolves from both shipped files) and `TestEveryTokenHasExactlyOneBinding`
  (the mapping covers the vocabulary exactly).
