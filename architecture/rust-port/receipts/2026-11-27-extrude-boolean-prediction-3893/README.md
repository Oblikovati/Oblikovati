# Extrude Boolean prediction

The published scenario uses the 3 mm circle on the existing solid's
top face. Advanced Settings shows Predict Boolean Operation checked
when the draft opens. Clearing it keeps Default direction when the
user chooses Cut. The top face remains filled until the user
explicitly chooses Flipped. The failed outward Cut status clears when
the corrected preview succeeds.

The Flipped Cut shows the circular opening and keeps one body. The
desktop draft records Cut and Flipped with the same document generation.
The default Cut scenario checks that prediction still selects Flipped
and that accepting the opening supports Undo and Redo. The selector
scenario verifies Icons and Dropdown still preserve an explicit Join.
[Input checksums](published-inputs.sha256).

All three published cases passed on Wayland and X11, 23 steps per
desktop. [Wayland manual](wayland.xml), [default](wayland-default-cut.xml),
[selector](wayland-selector.xml). [X11 manual](x11.xml),
[default](x11-default-cut.xml), [selector](x11-selector.xml).

The default Cut regression also passed against pinned scenario
snapshot `b576445b` with nine steps on each backend.
[Wayland pinned](wayland-pinned-cut.xml), [X11 pinned](x11-pinned-cut.xml).
