# Sketch Text anchor degrees of freedom

Both native tests passed on Linux X11 under Xvfb with Vulkan Lavapipe,
API v0.165.0, and Martensite v0.13.1. The tested tree
`10ea23834714598ccc03ef40f5b9696971e1b09a` matches application commit
`d79e6195f25f15f238341a7e9be8b3768f864a61`. The corresponding Sketch Text feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

| Scenario | Before | Native result | Source expectation |
| --- | ---: | ---: | ---: |
| `text-01` | 210 | 212 | 215 |
| `text-13` | 28 | 30 | 33 |

The tests use generated native sketches and reopen them through File Open.
They use a single insertion location because the source supplies no saved
sketch or alternate placement gesture for these captures. Both results
add one Text object and one free anchor point. The tests compare the
entity IDs and all existing constraints before and after creation.

- [210-dimension Part1](text-01/Part1.opd) starts in Sketch1, Front view.
  Place `a` at (450, 650) in a 1440 × 900 client window. The test selects
  6.10 mm and checks the accepted glyph and
  [Ready with 212 dimensions needed](text-01/text-point-ready-212.png).
  Part SHA-256: `484e0470a4d3ad6b8f2ae8e4f7fc02d4b44db74514761e72c18319f2d8c04b19`.
- [28-dimension Part1](text-13/Part1.opd) starts in Sketch1, Front view.
  Place `ab` at (900, 580). The isolated font fixture supplies Tahoma at
  3.05 mm. The test selects both characters and checks the
  [Bold preview](text-13/text-bold-creation-preview.png). Undo restores
  the whole 28-dimension sketch. [Redo](text-13/text-bold-redo-30.png)
  restores the same anchor, glyph pixels, and 30 dimensions needed.
  Part SHA-256: `57676a73636ecda432cdb32e717f106461a6f73648796c7860283c0cc6dced64`.

The Tahoma run uses the same font archive and checksum as the
[saved Bold case](../2026-11-21-sketch-text-tahoma-3907/README.md). Go
Sketch Text tests passed at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
The Go Text tool uses a by-reference TextBox with an anchor record.
It provides no basis for the three additional freedoms in these cases.

These are native replays. See the
[scenario discrepancy](../../scenario-defects/text-creation-degrees-of-freedom.md).
