# Sketch Text Bold preview and Cancel

Pinned scenario `text-11` passed all eight steps on the Rust desktop at
`e7d8ff03e1ca1897147294d827a6eed135b701df` with published API v0.161.0.
The runner restored a clean scenario snapshot at
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The fixture created a DejaVu Sans `c` in an XY sketch through the desktop.
X11 pointer events reopened the text. Ctrl+A selected the content. The Bold
button became active, the editor glyph gained foreground pixels, and the
canvas outline changed. Cancel returned to Ready with the same text entity
and solver count. The analytic glyph test also measured a greater native face
area for bold `c` than regular `c` at the same size. Bold currently requires
the full selection pending [Martensite #576](https://github.com/Oblikovati/Martensite/issues/576).

```sh
OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-11 XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The run used Xvfb and Vulkan Lavapipe. Set `VK_DRIVER_FILES` to the installed
Lavapipe manifest to use the same renderer. Windows and macOS UI evidence
is pending. The run supplies an R0 record. Strict verification also requires
bounded build and runtime inputs in [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).

The Go reference at `aa8297d1b0ac33c024f19e54821536d76e8a447a`
passed its glyph profile, anchor translation, saved text style, and typed
add/edit/get tests in `model/sketch` and `addin/router`. Those Go tests
supply geometry and transport references. The Rust desktop test supplies
the Bold control and Cancel evidence.
