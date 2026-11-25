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
- The application bar displays the brand mark. The ribbon and document chooser
  use the matching SVGs where the handoff supplies artwork for a shipped
  command. Navigation uses 16 px wheel, pan, zoom, and orbit glyphs and
  chevrons inside a 28 px bar.
- Inter is the default UI font. Numeric fields in Extrude and status readouts
  use JetBrains Mono. Sketch Text keeps its existing authored text faces.
- Extrude uses 40/60 field alignment for the sketch, profile, primary distance,
  and extent controls. The default floating width is 280 px. The Input
  Geometry and Behavior sections open by default. A new solid uses the
  translucent teal preview color.
- Default and Presentation use the dark and light studio gradients. High
  Contrast uses a solid black viewport. Theme selection and viewport colour
  scheme remain separate preferences.

The existing theme API names and custom XML snapshots still load. If a saved
custom theme is already called High Contrast, it keeps its name and the
built-in gets a distinct label.

## Evidence

[Native tests and captures](receipts/2026-11-25-visual-identity-4063/README.md)
show one Part in the three theme and viewport palette pairs,
sample the navigation and dock surfaces, and save the frames. Additional
checks cover live Extrude preview and validation, native Pan and Orbit input,
icon rasterization across display densities, and theme persistence.
