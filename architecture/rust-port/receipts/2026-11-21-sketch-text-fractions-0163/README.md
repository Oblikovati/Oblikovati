# Sketch Text stacked fractions

The pinned Sketch Text scenarios used native keyboard and mouse input
under Xvfb and Vulkan Lavapipe. The runner checked the clean snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

- `text-28` stacked the original editable `1/2` and showed a visible blue
  separator between the digits. Accept kept the original text ID and
  insertion point at 35 dimensions needed.
- `text-30` saved preset `f` from an existing fraction and left that
  entity intact on Cancel. Applying the preset to a new placement restored
  the blue fraction, Arial, 6.10 mm, and width 50. Accept created a second
  entity and anchor at 37 dimensions needed.
- `text-36` rejected zero width in a separate warning and kept the glyph.
  After acknowledgment it selected the zero for replacement. The editor
  accepted 50 without a second warning, and Cancel kept the accepted text.

The tested application sources are in `5b06361916e3ba680368831a587bbf5070687a14`.
They use [API v0.163.0](https://github.com/Oblikovati/Oblikovati.API/releases/tag/v0.163.0) at
`c051de97568c0345921f0f900982da31eb9ddca4` and Martensite at
`bef25f47548431c8f128f08f66a1f719bcee4982`.
[Stacked fraction API change](https://github.com/Oblikovati/Oblikovati.API/pull/319).

The outline checks confirm that the digits bracket the separator, that
width 50 halves the separator without moving the rows, and that the glyph
can be picked. Typed and native YAML tests preserve the source range
and insertion point through editing, undo, and reopen. The preset run
used genuine Arial in an isolated Fontconfig fixture.
See the [Arial fixture record](../2026-11-21-sketch-text-arial-3907/README.md).

For a focused text run, select the scenario by name. Set
`VK_DRIVER_FILES` to the installed Lavapipe manifest. For `text-30`,
also set `FONTCONFIG_FILE` to a configuration that can read Arial.

```sh
env OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-30 XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

Build and runtime inputs were unsealed. Strict verification remains open
under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
