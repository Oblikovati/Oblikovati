# Provenance — Quick Access Toolbar icons (PR-5a)

This file records the source and licence of every third-party-derived asset in this
change for maintainer review and distribution compliance.

## Summary

The five SVG glyphs added by this PR (`file-menu.svg`, `open.svg`, `save.svg`,
`undo.svg`, `redo.svg`) are **derived from Lucide icons**, an open-source icon set
released under the **ISC License** (GPL-2.0-compatible permissive licence). The
original glyphs authored for this PR were replaced because their provenance was not
documented; the replacement glyphs are adapted from a primary upstream source with an
explicit permissive licence. **No Autodesk artwork, source, or proprietary assets are
included.**

## Source

- **Project**: Lucide (lucide-icons/lucide) — https://github.com/lucide-icons/lucide
- **Licence**: ISC License (see full text below). The Lucide set also contains icons
  derived from Feather (MIT, © Cole Bemis); **none of the icons used here are in the
  Feather-derived list** published in the Lucide LICENSE file.
- **Version / commit**: tag `1.33.0` = commit `59978cecf84986af59f1f9f503bcebdc89c6d166`
  (2026-08-19).
- **Access date**: 2026-08-21.
- **Source files** (raw at `https://raw.githubusercontent.com/lucide-icons/lucide/1.33.0/icons/<name>.svg`):

  | Oblikovati asset | Lucide source icon | SHA-256 of Lucide source |
  | --- | --- | --- |
  | `head/icon/assets/file-menu.svg` | `menu.svg` | `af6491988ac1ba30dc5414f485bc48713e957e1edb0fbef4a0dd8691c3c21598` |
  | `head/icon/assets/open.svg` | `folder-open.svg` | `ee57452b40e71df310214091dd9da1941a397c83197b8ff8d7ae66dd4242578b` |
  | `head/icon/assets/save.svg` | `save.svg` | `ee9d56a7fec4b20dd6689546d41f68219ea8cbd67f99bd17a1bdaff5be6edb53` |
  | `head/icon/assets/undo.svg` | `undo-2.svg` | `efad2edbbd2be6038cc23fd64df2d012fad5cc6460dfbf227cfa3d990e9e897f` |
  | `head/icon/assets/redo.svg` | `redo-2.svg` | `4cbd2cdebc70b6b7979c681ed3f313d9cbb102465dcf7d0d2d6a72c937b344bc` |

## Modifications made

The Lucide sources are 24×24 stroke-based glyphs (`stroke="currentColor"`,
`stroke-width="2"`). Adaptation to the Oblikovati icon contract (ADR-0033) was
**geometry-only**:

1. **Background plate**: added the repo-standard rounded plate
   `<rect x="1" y="1" width="22" height="22" rx="4" fill="#00ff00" stroke="none"/>`
   (the ADR-0033 background sentinel), which Lucide glyphs do not carry.
2. **Role sentinels**: replaced `stroke="currentColor"` with the ADR-0033 primary
   sentinel `#000` on the main linework. For `undo.svg` and `redo.svg`, the
   direction-arrow path is painted with the secondary sentinel `#ff0000` (the
   action/result element role), matching the repo's convention of accenting the
   action element of a glyph.
3. **Source geometry was retained**: every Lucide path/rect element is preserved
   byte-for-byte apart from formatting and role-colour substitutions; the added
   background plate is the documented ADR-0033 adaptation (verified against the
   SHA-256 hashes above).

## Attribution

- Lucide icons are © Lucide Icons and Contributors, ISC License.
- Feather-derived icons in the Lucide set are © Cole Bemis, MIT License (not used here).

## Licence notice (ISC, required by the licence)

The ISC licence requires that the copyright notice and permission notice appear in all
copies. The full licence text is reproduced below and applies to the derived glyphs:

```
ISC License

Copyright (c) 2026 Lucide Icons and Contributors

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
```

## Legal note for the PR description

> This change contains independently authored implementation code; no Autodesk
> source, binaries, proprietary documentation, or Autodesk artwork are included.
> Compatibility references are descriptive only. The icon glyphs are derived from
> Lucide (ISC License); see LEGAL/PROVENANCE.md for full provenance.
