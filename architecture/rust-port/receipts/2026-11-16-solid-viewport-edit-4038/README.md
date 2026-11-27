# Solid viewport feature edit diagnostic

Base `934c80d38`, Martensite `bef25f47548431c8f128f08f66a1f719bcee4982`,
Go reference `aa8297d1b` (`app/feature_edit.go` and `app/feature_edit_test.go`).
The live scenario file is retained as `source.feature.gz`. Its SHA-256 was
`34abe4989ff7020bb1a3a7a5207f551b96535cfbbb33c8212e1da77ecfa503aa` before
and after the runs. It is unversioned live source, not the pinned snapshot.

On Linux, Xvfb and Lavapipe, X11 XTEST clicks opened the feature context menu
and editor, and XTEST keys changed the distance. Fixture creation used the
existing scripted test helper. The accept scenario passed 10 steps and checked
one feature, a visible changed preview, and at least four analytic 20 mm edges.
The cancel scenario passed six steps and checked at least four 10 mm edges and
the visible retained body. Screenshots were visually inspected. The selected
solid screenshots have very dark faces and a triangular shading boundary; their
cause and correctness have not been established by this diagnostic (#4044).

The existing edit, cancel, undo, redo, and reopen regression also passed.

The full feature run passed two of five scenarios. Three stopped at unbound
creation and picking steps. The source still requires fixtures and captures,
and build provenance is open (#3978). These runs are diagnostics, not verified
BDD acceptance. The viewport window was retired after each run.
