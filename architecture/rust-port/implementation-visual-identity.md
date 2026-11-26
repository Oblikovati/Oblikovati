# Desktop visual identity

The Rust desktop uses the designer's Ultra-Minimal Technical identity from
`MCAD-Scenarios/Design`. The design handoff's token JSON defines the built-in
Dark, Light, and High Contrast colors. The bundled SVG artwork and facet paint
values come from the same handoff.

## Native integration

- The shared ImGui chrome, including detached drawing windows, uses semantic
  surfaces and the active theme's accent. Window headings use neutral fills.
  Selected ribbon and document tabs have an indigo underline.
- The Model, Material Browser, and Appearance Browser use the same chrome when
  docked. Theme changes schedule the frame needed to refresh native dock tabs.
  A fresh Model dock takes about a quarter of a regular desktop workspace.
  The initial width stays within the dock budget and caps at 480 logical px.
  When the Part workspace narrows, side docks preserve their 180 logical px
  minimum where it fits alongside the reserved viewport width. Widening the
  workspace restores the user's preferred dock sizes. The last resized dock
  gets priority within the available width, including after a restart. The
  viewport width limit also applies while the user holds a splitter.
  Docked asset selectors keep the caption on the left and shorten the
  value before the dropdown arrow. Inactive dock tabs use readable text
  with the accent underline reserved for the active tab. The Model browser’s
  search field encloses its clear control, sketch badges use 10 px mono,
  and the dashed rollback marker uses the theme’s strong border color.
- At the native window minimum, compact ribbon groups share the row
  using their measured captions. Long captions shorten before the
  disclosure arrow and keep the full group name in the tooltip.
- The camera projects into the live document area as the window and docks
  resize. Layout changes preserve the current camera and Previous View
  restores its saved camera in the current layout. A cropped origin plane
  keeps its projected label inside the visible part of the same surface.
  The label backing follows that projection and fades with the text.
  The navigation bar moves alongside the ViewCube when the viewport
  is too short to stack them. ViewCube face labels use heavier glyphs so
  their strokes remain distinct when the face foreshortens. Their scale
  is capped by the width of each face. The picker calibrates its warm fill to the viewport backdrop before
  the renderer blends it in linear light.
- Hole uses the shared left-aligned property labels and outlined numeric
  fields. An empty Depth field displays Through all. The floating panel
  fits the rows and action footer. Offset Plane uses the same property
  layout and grows to fit wrapped validation messages. Move Bodies aligns
  the body selector and all three offsets in the shared value column.
  Shell fits its face count, outlined Thickness field, and footer to the
  available viewport.
- Home Recent leaves 12 logical pixels of scroll-end padding above the
  document tab strip in both List and Grid views.
- The application bar displays the brand mark. Mapped ribbon commands and
  document chooser entries use the handoff SVGs. Navigation uses 16 px
  wheel, pan, zoom, and orbit glyphs and
  chevrons inside a 28 px bar. The Tools Options and Normal Sketch Create
  flyouts use 28 px full-width rows with 16 px leading icons and compact
  Pin footers.
- The Sketch and 3D Model ribbons use additional handoff icons for draw
  tools, constraints, work features, and model operations. The rectangle and
  arc variants use their own artwork, and sketch fillets use a different
  glyph from solid fillets. The Precise Input icon draws its lettering as
  vector paths so it renders without system fonts.
- Inter is the default UI font. Numeric fields in Extrude use JetBrains Mono,
  and status telemetry uses it at 10 px. Fully Constrained uses the active
  theme’s success color. Sketch Text keeps its existing authored text faces.
- Extrude uses 40/60 field alignment for the sketch, profile, primary distance,
  and extent controls. The default floating width is 280 px. The Input
  Geometry and Behavior sections open by default. A new solid uses the
  translucent teal preview color.
- The Default colour scheme follows the studio backdrop of the active theme.
  Named colour schemes and explicit background modes take precedence.
  Viewport annotations choose contrasting ink when a scheme overrides the
  backdrop. The status bar uses the active chrome palette.

The existing theme API names and custom XML snapshots still load. If a saved
custom theme is already called High Contrast, it keeps its name and the
built-in gets a distinct label.

## Evidence

[Native tests and captures](receipts/2026-11-25-visual-identity-4063/README.md)
show one Part in the three theme and viewport palette pairs,
sample the navigation and dock surfaces, and save the frames. Additional
checks cover live Extrude preview and validation, native Pan and Orbit input,
icon rasterization across display densities, and theme persistence.

The [ribbon icon checks](receipts/2026-11-25-identity-ribbon-icons-4063/README.md)
include the rendered Sketch ribbon and the 3D Model ribbon in all three
palettes.

The [design-gap audit checks](design-gap-audit.md) track the remaining desktop
refactor and verification against the designer's Windows audit.
