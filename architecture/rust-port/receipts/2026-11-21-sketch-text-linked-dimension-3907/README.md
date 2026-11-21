# Edit a dimension linked to Sketch Text

Pinned `text-17` passed all six steps on Linux X11 under Xvfb with Vulkan
Lavapipe, API v0.165.0, and Martensite v0.13.1. The tested tree
`f8f24378658dcfd75c65c7c7217dcbd428084e17` matches application commit
`f54b80f51a03c7a8ab51857e6dc34bf1e4615ea8`. The clean scenario snapshot was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The test creates a YZ sketch with a driving 10 mm dimension. It uses Format
Text to insert `d0` with the default three decimal places. It opens the
visible dimension with a double-click, confirms that the editor identifies
`d0`, and presses 1, 2, and Enter. The source annotation shows `12` and the
linked Text shows `12.000`. Ctrl+Z restores the authored sketch, the source
annotation at `10`, and the linked Text at `10.000`.

See the native captures [after the edit](linked-dimension-12.png) and
[after Undo](linked-dimension-undo-10.png). The Go Sketch Text and glyph
tests passed at `aa8297d1b0ac33c024f19e54821536d76e8a447a`. The report
supplies R0 baseline evidence. Strict verification remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
