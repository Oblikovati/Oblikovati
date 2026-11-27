# Sketch Text line spacing and typed operations

Pinned scenario `text-21` passed all seven steps with native X11 keyboard
and mouse input under Xvfb and Vulkan Lavapipe at application commit
`57d8e36962bf77666e7be44699c6822e1e316465`. The scenario revision was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`, and the feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

Center and Middle changed the rendered placement. The spacing list marked
Single and offered Double, 1.5 Lines, Multiple, and Exactly. Selecting Double
gave twice the previous separation between the two visible editor rows.
The native screenshot shows the selected settings.

The application uses the released API v0.162.0 at
`f489eb15d33bb4fea751bf230a92a65ccafd9dec` and Martensite at
`bef25f47548431c8f128f08f66a1f719bcee4982`.
[API release v0.162.0](https://github.com/Oblikovati/Oblikovati.API/releases/tag/v0.162.0).

The typed host tests cover the original text ID and insertion anchor,
spacing and content through an edit, undo and redo, explicit spacing reset,
rejection of invalid edits, and native YAML reopen. The outline tests
cover natural, double, and exact spacing at the same glyph scale.

```sh
env OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-21 XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

Set `VK_DRIVER_FILES` to the installed Lavapipe manifest for this Linux
run. Build and runtime inputs were unsealed. Strict verification is tracked
under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
