# Native layout and viewport audit checks

Linux X11, Openbox, and Vulkan Lavapipe ran the native checks from
`5949cb612` with the design audit fixes. The worktree included existing
document-opening and saving edits.

- A pointer drag reduced Extrude to less than 170 px tall. The properties
  body scrolled independently, and the user could still see OK and click
  Cancel. Switching between Chamfer and Extrude closed the previous draft
  in either direction. [Short Extrude](Extrude.png)
- At 1440 px, the Sketch ribbon showed named compact groups, both commands
  on the last Create row, and the emerald Finish Sketch. Pointer input
  finished the sketch. The published Ribbon Appearance scenario passed all
  18 steps through Small, Compact, Large, and Normal.
  [Sketch ribbon](Sketch.png) · [Cucumber](cucumber.json) · [JUnit](junit.xml)
- The native origin-plane picker showed all three planes in the viewport.
  Opening the picker preserved the sampled ViewCube face and navigation
  icon pixels, and clicking the projected XY label entered the sketch.
  [Origin-plane selection](Origin.png)
- The two extrusion-resume regressions passed. They verified the preview
  after a free or constrained sketch and its removal on Cancel. The
  screenshot check requires a rendered teal patch outside GUI windows.
