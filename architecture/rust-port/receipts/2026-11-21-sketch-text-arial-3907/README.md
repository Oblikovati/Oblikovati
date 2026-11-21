# Sketch Text Arial and height preview

Pinned scenario `text-18` passed all five steps on Linux X11 with Xvfb and
Vulkan Lavapipe at application commit
`4ea876105ade0874f3b753b8984e82807472925d`.
The runner checked the clean source snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8` before and after the run.
The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The setup created bold italic DejaVu Sans text, reopened it, and selected
both characters through X11. Selecting Arial changed the glyph pixels and
kept the existing style. The height field showed 6.10 mm after selection.
The visible outline grew from 11 to 21 pixels high at the same view scale.
Cancel kept the original text and sketch geometry.

The run used the Microsoft Core Fonts Arial archive with SHA-256
`85297a4d146e9c87ac6f74822734bdee5f4b2a722d7eaa584b7f2cbf76f478f6`.
The four TrueType files matched the installer checksums. Fontconfig resolved
the Bold Italic face to `arialbi.ttf` in the isolated fixture directory.
[Microsoft Core Fonts archive](https://sourceforge.net/projects/corefonts/files/the%20fonts/final/arial32.exe/download)
and [Debian installer checksums](https://sources.debian.org/src/msttcorefonts/3.8.1/update-ms-fonts).

To repeat the focused run, point `FONTCONFIG_FILE` at a configuration with
the Arial and DejaVu Sans fonts. Use the installed Lavapipe manifest for
`VK_DRIVER_FILES`.

```sh
env OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-18 XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

Go reference `aa8297d1b0ac33c024f19e54821536d76e8a447a` provided the
glyph and recipe baseline. See the
[Go test evidence](../2026-11-20-sketch-text-italic-3907/README.md).
The runtime and build inputs were unsealed. Strict verification remains open
under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
