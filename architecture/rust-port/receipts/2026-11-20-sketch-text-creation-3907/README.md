# Sketch Text creation and empty acceptance

The native Linux window passed four focused scenarios and 31 steps against
snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature hash matches the
live Text source. Text actions used X11 mouse and keyboard input. The native
window was 1440 × 900 under Xvfb with Vulkan Lavapipe.

- [`text-02`](text-02/cucumber.json) closes an empty editor while leaving the
  placement prompt and 28 degrees of freedom. The sketch remains unchanged.
- [`text-04`](text-04/cucumber.json) checks that one Undo removes the creation
  and Redo restores the geometry, rendered text and solver count.
- [`text-07`](text-07/final.png) checks preview, accepted placement, Undo and
  Redo. The recorded pixel `(450,650)` reaches the sketch in this window.
  Undo removes the glyph pixels. Redo restores the original model identity,
  placement and visible glyph.
- [`text-12`](text-12/final.png) accepts an empty editor and reopens it at the
  same insertion point. It opens empty and does not add text to the sketch.

The two empty-content scenarios use a generated native sketch with the
required 28 degrees of freedom. Its seven free lines are unrelated to Text.
The history cases use an empty native sketch and compare the accepted glyph
against a framebuffer taken before Text creation. They use a single Undo and
Redo and compare the original text entity, anchor, style, and history.

The focused runs use `sketch_scenarios_use_the_native_window` with
`OBK_BDD_FEATURE=text` and each `OBK_BDD_SCENARIO` filter. The two previous
native Text regressions also passed. Pinned Arc Three Point history and
Viewport Orbit OK checks passed, which cover the shared step bindings.
The pinned cases are R0 baseline records. Strict acceptance still needs
bounded build and runtime inputs under #3978.

The full live Text run covered all 38 declarations. Seven passed and 31
failed at undefined steps. See the [Cucumber report](cucumber.json). The run
used `OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios` and
`OBK_BDD_FEATURE=text` with no individual scenario filter. The feature and
its supporting inputs had the recorded hashes before and after the run.
The scenario owner's `BLOCKERS.md` notes were updated after the run. The
feature inventory uses the scenario and supporting specification hashes.
