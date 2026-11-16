# Prism presentation orientation diagnostic

Base `934c80d38`, Martensite `bef25f47548431c8f128f08f66a1f719bcee4982`.
A temporary trace on the copied extrusion scenario showed the stored Start
lineage was Forward while the matching face in the copied solid was Reversed.
The host emitted +Z normals on both caps. A separate counterclockwise square
had all four side lineages Forward while their body faces were Reversed. The
outward-winding regression failed on that square before the host change.

The host resolves each lineage face to the unique face in the body before
meshing it. The final focused run passed 36 prism tests, including positive
and oblique negative sweeps, face identity and refusal of a foreign face.
The named-face plane integration test also passed through edit, undo, redo,
and reopen, retaining the expected body-relative face frames.
The native Linux Xvfb/Lavapipe diagnostic rendered positive and negative
extrusions and checked presented pixels; both captures were visually inspected.
The native diagnostic was temporary and is not a portable acceptance test.
Its source is retained in `oriented-probe.rs.txt`; the log records the test.
Copy it to `crates/viewport/tests/native/oriented_probe.rs` and add the matching
module declaration to `crates/viewport/tests/native.rs` to repeat it in a
separate worktree. Use the Linux `native` test target under Xvfb.

The unselected `before.png` and `after.png` are pixel-identical (ImageMagick
absolute error count zero). Thus this corrects a face-orientation invariant,
but does not explain or fix the triangular shading observation in #4044.
The screenshot scenario used temporary logging and an unpinned live scenario;
it is diagnostic evidence, not verified BDD acceptance. Test windows were closed.
