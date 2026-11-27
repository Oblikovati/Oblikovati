# Full Ellipse run

Application commit `b6c4a99fd20910a3271cc6cc6817b1e08b80667d`. The published
Ellipse feature, supporting files, and seven native Part fixtures matched the
input manifest before and after the run.

The full feature ran all 19 scenarios in a native X11 window under Xvfb with
Vulkan Lavapipe. Scenario steps used scripted input. Nine scenarios and 104
steps passed. Ten scenarios failed. This diagnostic run returned Cargo exit
status 101. The published source had no Git revision.

The passing cases were `ellipse-01`, `03`, `06`, `07`, `08`, `12`, `18`,
`19`, and `20`. The rotated Measure case reproduced
[Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).
The remaining failures cover curvature display and settings, and axis-radius
dimension editing and measurement. The full feature still needs acceptance.

The `focused/` reports cover the later radius tests on the same published
feature hash. `ellipse-13` through `ellipse-17` passed individually. These
tests covered editing the major radius, undo and redo, cancelling and accepting
minor-radius edits, and measuring both radii on a second saved Part. The
last case also checked that closing Measure left the oval visible in the
native viewport. The focused runs used working-tree test changes and matched
the full-run input manifest before and after. The inventory uses the full
run's 9 of 19 result until another full run replaces it.
