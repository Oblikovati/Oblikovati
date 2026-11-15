# Ribbon icon source

Except for `template-document.svg` and `find.svg`, these SVG files are copied without changes from Go `head/icon/assets/` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The application source uses
GPL-2.0-only. No separate asset license or upstream attribution was found.

The four sentinel paints are role markers. They must be rasterized as separate
coverage masks and composed with the active icon theme tokens. Do not display
the sentinel paints directly. This set covers the current Rust ribbon commands;
it does not claim the full 281-icon Go catalogue is ported. The generic
`template-document.svg` is an Oblikovati Rust asset for document types without a
dedicated Go glyph.

The `find.svg` glyph is an Oblikovati Rust asset for the browser search command.
