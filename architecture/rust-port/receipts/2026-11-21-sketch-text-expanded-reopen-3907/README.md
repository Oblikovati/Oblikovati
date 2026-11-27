# Reopen expanded Sketch Text

Pinned `text-38` passed all nine steps on Linux X11 under Xvfb with Vulkan
Lavapipe, API v0.165.0, and Martensite v0.13.1. The staged test tree
`cdb9d554353f216d6cde5917eb838d8b3722ed6c` matches application commit
`062ee7327558391ea2bf92340b83baad0acd7394`. The clean scenario snapshot was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The fixture creates Sketch2 with a driving 10 mm dimension, two stacked
fractions, two numbered paragraphs, and two Text entities with linked
fields. Save As writes [Part6.opd](Part6.opd). The generated document's
internal title is Part1. The file-name field uses the saved file basename.
The fixture uses DejaVu Sans.

The test closes the part, opens its saved file in the same session, and
chooses Edit Sketch on Sketch2. Authored geometry and stable identities
match across reopen. The viewport shows the two fractions, `1. a`,
`2. b`, `Part6.opd`, `10.00`, and the earlier field at `10.000`.
Ready shows 47 dimensions needed.

The saved file SHA-256 was
`ab3ecb1e4553ecc8b58ee2738d759f72ad0ecdf85680f869ff51d7af16ade793`.
The Go Sketch Text and glyph tests passed at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The report supplies R0 baseline evidence. Strict verification remains
open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
