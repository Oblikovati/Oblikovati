# Sketch Text Italic history

Pinned scenario `text-15` passed all eight steps at application commit
`368fc5dee2ac56baa4dfeb1e7edd9cce0e1d4035` with published API v0.161.0.
The runner restored a clean scenario snapshot at
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The test created a planar sketch with 31 degrees of freedom and placed
bold DejaVu Sans `ab` at a free point. This gave the required starting state
of 33 dimensions needed. X11 input reopened the text, selected both
characters, and accepted Italic. The viewport showed the edited outlines.
Undo restored the original pixels and text entity. Redo restored the
edited pixels. Each state kept 33 dimensions needed. The text recipe
kept its content, anchor, height, and Bold setting.

```sh
OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-15 XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The focused native test
`native_italic_text_history_keeps_bold_content_and_anchor` also exercised
Ctrl+Z and Ctrl+Y through X11 input. Use the installed Lavapipe manifest for
`VK_DRIVER_FILES` to reproduce this Linux run. Windows and macOS UI evidence
is pending. Strict verification requires bounded build and runtime inputs in
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).

Go reference `aa8297d1b0ac33c024f19e54821536d76e8a447a` passed its glyph
profile, anchor translation, saved style, and typed add/edit/get tests in
`model/sketch` and `addin/router`. The Rust desktop tests cover the Italic
control and history.
