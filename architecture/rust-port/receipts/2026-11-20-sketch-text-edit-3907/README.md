# Sketch Text edit and history

The focused pinned Text runs passed three scenarios and 23 steps on the Linux
X11 desktop under Xvfb and Vulkan Lavapipe. The verified scenario snapshot was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The live Text feature has the same
SHA-256 as the pinned file. Use `live-inputs.sha256` to check the current live
feature and its support files.

- `text-08` covers a fresh text draft after Cancel. [Final frame](text-08/final.png)
- `text-09` covers an existing glyph edit and Cancel. The original `b` comes
  back, the sketch returns to Ready, and the solver count and entity count
  stay the same. [Final frame](text-09/final.png)
- `text-10` covers accepting `c` and undoing and redoing the edit at the same
  location. [Final frame](text-10/final.png)

The head regression `sketch_text_context_replaces_the_same_entity_and_restores_history`
used X11 mouse and keyboard input to check the original entity, anchor, and
history. The two viewport Text tests also passed. The preview test compares
its rendered pixels with an independent replacement-only frame and checks
that Cancel restores the original framebuffer.

The run used `sketch_scenarios_use_the_native_window`, `OBK_BDD_FEATURE=text`,
and the corresponding `OBK_BDD_SCENARIO` filter for each case. These are R0
baseline records and focused diagnostic results. Strict acceptance still needs
bounded build and runtime inputs under #3978.
