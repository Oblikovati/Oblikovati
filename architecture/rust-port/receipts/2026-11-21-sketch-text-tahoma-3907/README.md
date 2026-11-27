# Sketch Text Tahoma Bold after Redo

Pinned scenario `text-14` passed all five steps under Xvfb and Vulkan
Lavapipe at application commit
`518869c55147adbef3adcf1a5956d6c6a90003a8`. The source snapshot was `b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature
SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The fixture created bold Tahoma `ab` at 3.05 mm. Ctrl+Z removed the text
and Ctrl+Y restored the same object and glyph. A native double-click on
the glyph reopened Format Text. The rendered Bold button was highlighted
and the editor displayed the original font, height, and characters.
Cancel kept the committed glyph footprint, sketch recipe, and solver count.
The separate DejaVu Sans regression also checked that a single glyph click
and an empty-canvas double-click left Format Text closed.

The isolated Fontconfig fixture used the Tahoma fonts from Microsoft
`IELPKTH.CAB` with SHA-256
`c1be3fb8f0042570be76ec6daa03a99142c88367c1bc810240b85827c715961a`.
Only the two font files were extracted. Fontconfig resolved Tahoma Bold
to `tahomabd.ttf` from the fixture. The font remains an optional test
input. [Microsoft Tahoma family](https://learn.microsoft.com/en-us/typography/font-list/tahoma).
[Winetricks archive checksum](https://github.com/Winetricks/winetricks/blob/master/src/winetricks).

To repeat the scenario, set `FONTCONFIG_FILE` to a configuration that can
read Tahoma and `VK_DRIVER_FILES` to the installed Lavapipe manifest.

```sh
env OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-14 XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The targeted Go tests for glyph profiles, anchor placement, style
round-trip, and typed add, edit, and get passed at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The build and runtime inputs
were unsealed. Strict verification remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
