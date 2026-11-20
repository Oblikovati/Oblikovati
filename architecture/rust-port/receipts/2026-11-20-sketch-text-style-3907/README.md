# Sketch Text font, height, and alignment

The native X11 regression selects an installed font through the font filter,
chooses the 6.10 mm preset, and sets Center and Middle alignment. It uses the
same font registry as the viewport. The test compares glyph pixels after each
change. [Preview capture](format-preview.png)

Cancel restores the original glyph and model. OK preserves the text identity
and anchor. Undo and Redo restore both states. The test saves the native part,
closes it, opens it again, enters Edit Sketch from the browser, and opens Format
Text from the glyph. It checks the saved style and model against the accepted
state. Both earlier native Text regression tests also passed.

The tests used Xvfb at 1920 × 1080, a 1440 × 900 desktop window, and Vulkan
Lavapipe. The installed-font choice was DejaVu Sans Mono.

The unfiltered live Text run covered all 38 declarations. Seven passed and
31 stopped at undefined steps. The feature and supporting files retained
their hashes throughout the run. [Cucumber report](cucumber.json)

The full run used `sketch_scenarios_use_the_native_window` with
`OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios` and `OBK_BDD_FEATURE=text`.
Strict acceptance still needs bounded build and runtime inputs under #3978.
