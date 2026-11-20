# Sketch Text height validation

The native X11 regression opens a committed glyph for editing, selects the
6.10 mm height preset, replaces it with zero, and presses Tab. A warning asks
for a value greater than 0.01 mm. The editor leaves the last valid preview
active. After OK, the previous glyph pixels return undimmed and the height
field selects the invalid value. Typing 6 and pressing Tab accepts 6 mm in
the draft. Cancel restores the original committed sketch.

[Warning capture](height-warning.png)

The two control tests also cover the exact lower boundary, negative and
nonfinite values, and repeated validation before the modal opens. The native
run used Xvfb, a 1440 × 900 window, X11 mouse and keyboard events, and Vulkan
Lavapipe. It exercised the height workflow with a plain text fixture. The
published `text-37` case also requires the recorded fraction and solver count.

The unfiltered live Text run passed seven of 38 declarations. The other 31
stopped at undefined steps. The run used the same application code, before
the focused height regression was updated to compare pixels after dismissing
the modal. The scenario and supporting specification hashes stayed the same.
The scenario notes changed after the run to record the `text-37` question.
[Cucumber report](cucumber.json)
