# Native ribbon icon integration

This work continues [#3798](https://github.com/Oblikovati/Oblikovati/issues/3798)
and the independent icon factor in
[#3850](https://github.com/Oblikovati/Oblikovati/issues/3850). The target remains
Martensite v0.11.0 at `4141137449d911884c9a3fd261f7913498548e90`.

## Owned texture boundary

Each `GuiBatch` retains a portable `GuiTextureKey`. Its `GuiFrame` owns an
immutable resource bank, including the exact font generation and color images.
Snapshot creation resolves every texture ID before copying indices. Null,
unknown, duplicate, and reserved bindings are errors. Empty batches do not bypass
binding checks. The font resource still requires white RGB with alpha coverage;
color images have their own resource class.

The compiler converts straight-alpha sRGB bytes to the SDK's linear RGBA image.
It retains one compiled image per binding and refuses a frame whose resource
bank differs. Older frames retain their original pixels after a new bank or GUI
context destruction. The old font-only constructors delegate to this same path;
they do not provide a second renderer or an unknown-texture fallback.

Three resource-bank tests, eleven renderer tests, and two actual ImGui image
snapshot tests passed. The renderer checks include two distinct colored images,
straight alpha, sRGB conversion, exact clip regions, empty unknown batches, and
resource-generation refusal. The new offscreen color-image test explicitly
enables Vulkan validation and requires an empty observation result.

## SVG role core

The 27 copied command icons come from Go `head/icon/assets/` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The asset directory records provenance.
This is the current supported command set, not a claim that all 281 Go icons or
commands have been ported.

The project-owned raster boundary contains resvg 0.49.0, with default features
disabled. It uses the existing image and XML package versions. No previously
locked package was removed or upgraded; the SVG dependency adds 20 packages.
SVG parsing admits self-contained vector shapes, bounded source length,
element count, and nesting. It refuses text, external images, style-dependent
content, foreign namespaces, document types, and non-role paints before a role
image can be published. Image resource resolvers cannot read files.

Inherited SVG paints select Go's background, tertiary, secondary, and primary
coverage masks. All passes use one content box, alpha threshold 16, 4× sampling,
and a 92% longest-side target. The compositor uses the active role colors in
source-over order and returns straight-alpha sRGB. Seven tests passed, including
all 27 assets at Go's 18 px and 40 px sizes, layer alignment, recoloring,
transparent pixels, and named declines.

## Required UI work

The core is implemented; the ribbon does not display the new icons yet. The
production icon cache, command bindings, measured column layout, independent
icon controls, and live acceptance remain to be connected. Interim Clippy reports
unused icon/image functions until those production callers are in place. No PR
or completion claim is made for this unfinished change.

Use the Go rendering definitions, not fixed text-button grids:

- Large icon buttons each occupy a full-height column with centered captions,
  wrapped at word boundaries to at most two lines.
- Small and compact buttons fill columns top-to-bottom, at most three per column.
  The ten supported constraints use compact style in Go's authored order.
- Rectangle, Circle, and Arc variants are text popup choices. Go's default Arc is
  the three-point arc. The current extra rectangle buttons must remain reachable
  as variants when their duplicate top-level positions are removed.
- Icon raster size is `round(base × icon factor × backing density)`, with bases
  18 and 40. With the current text-scaled GUI coordinate system, divide physical
  icon dimensions by `request.layout_scale()` so text edits cannot enlarge icons.
- Compute grid and band height from current icon, font, padding, and spacing
  metrics. Panel footers share a baseline. Expanded and collapsed content call
  the same measured button and column-packing path.

References are Go `head/ui/chrome_ribbon.go`, `ribbon_caption.go`,
`ribbon_collapse.go`, `qat.go`, `app/commands_sketch.go`, and
`app/sketch_constraint_tools.go`. Native acceptance must cover independent
(text, icon) factors (1,1), (2,1), (1,2), and (2,2), command activation, overflow,
preview/cancel, and MCP captures. Run the full gate and cross-target checks before
first push; preserve the open profile/Extrude and ARM64 runtime issues.
