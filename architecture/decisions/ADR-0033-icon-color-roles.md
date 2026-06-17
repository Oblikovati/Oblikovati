# ADR-0033: Icon color roles (primary / secondary / tertiary / background)

- Status: accepted
- Date: 2026-06-11
- Issue: Oblikovati#655 (theme migration follow-on)
- Builds on: ADR-0021 (semantic tokens), ADR-0032 (Blender theme file format)

## Context

Ribbon icons were monochrome: each SVG rasterized to a white alpha mask, tinted with
the single `icon.tint` token at draw time. One color cannot distinguish an icon's
subject from its action (the arrow in extrude, the rounded corner in fillet), its
reference detail (anchors, construction lines), or give the glyph a backplate — and
the Blender theme files we now ship as defaults carry a richer per-category icon
palette we had no way to use.

## Decision

1. **Four color roles per icon**: `primary` (main linework), `secondary` (the
   action/result element), `tertiary` (supporting/reference detail), `background`
   (a rounded plate behind the glyph). The `icon.tint`/`icon.disabled` tokens are
   replaced by `icon.primary`, `icon.secondary`, `icon.tertiary`,
   `icon.background` — removed outright, not deprecated (alpha, no installed
   add-in base). Disabled buttons keep dimming via Dear ImGui's `BeginDisabled`
   alpha, which needs no dedicated token.
2. **Roles are authored with sentinel paints** in the SVG sources — placeholders the
   theme replaces, never shown on screen:

   | Role | Sentinel |
   | --- | --- |
   | primary | `#000000` (the pre-existing monochrome art reads as primary unedited) |
   | secondary | `#ff0000` |
   | tertiary | `#0000ff` |
   | background | `#00ff00` |

   Every asset must use only sentinel paints, and must have a plate and primary
   content (`TestAssetsConformToColorRoles` fails the build otherwise).
3. **One coverage mask per role, composed on the CPU.** `icon.RasterizeRoles` filters
   the SVG element tree per role (effective stroke/fill resolved through
   inheritance), renders each subset alone, and normalizes all passes with the ONE
   content box measured from the full glyph, so layers stay registered.
   `RoleMasks.Compose` layers the masks (background → tertiary → secondary →
   primary, straight-alpha src-over) with the theme's icon colors into the texture
   the head uploads. ImGui draws it with an identity tint.
4. **Theme changes re-compose, never re-rasterize.** The head's icon cache keeps the
   masks (theme-independent) and rebuilds only the composed textures when the theme
   revision changes — lazily, per icon actually drawn, which keeps the appearance
   editor's live preview cheap. Replaced textures are retired for a few frames
   before `DestroyTexture` (swapchain frames in flight).
5. **Colors map from the Blender theme files** (extending the ADR-0032 table) using
   Blender's own icon palette:

   | Token | Blender source |
   | --- | --- |
   | icon.primary | `ui@icon_collection` (the neutral icon gray) |
   | icon.secondary | `ui@icon_object` |
   | icon.tertiary | `ui@icon_modifier` |
   | icon.background | `ui@panel_sub_back` (translucent backdrop; alpha = plate opacity) |

## Consequences

- A theme recolors the whole glyph set four ways: dark ships neutral-gray linework
  with mint accents, blue detail, and a faint dark plate; light ships dark-gray
  linework with orange accents on a faint light plate.
- Adding an icon means drawing with the four sentinels; the conformance test rejects
  drift (foreign paints, missing plate/primary).
- The SVG role filter reuses the generic `theme/blenderxml` element tree; the icon
  package remains the only importer of the SVG rasterizer.
- Add-ins reading theme colors over the wire see the four `icon.*` tokens in the
  palette map like any other token.
