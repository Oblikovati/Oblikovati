# Extrude history, measurement and parameter preview

`extrusion-06` passed all six steps on Wayland and X11 using a disposable copy
of the published `revolve-profile.opd`. Accepting 8 mm made one feature, Undo
removed it, and Redo restored one solid with two edges that Measure displayed
as 8.000 mm.

The named-parameter check verifies a changed mesh and changed visible
preview pixels when its value changes from 10 to 20 mm. The earlier check
searched for orange pixels after the preview color had changed to cyan.
The [X11 suite](native-x11.txt) passed all 14 tests, and the focused
parameter check passed on both [Wayland](wayland-parameter-preview.txt)
and [X11](x11-parameter-preview.txt).

The Linux build used `8451de3e8` and the [test source changes](test-inputs.sha256)
with Martensite v0.14.2. Wayland ran on COSMIC. X11 ran under Xvfb with
Lavapipe. The [published source and fixture hashes](current-inputs.sha256)
matched before and after the runs.

- [Wayland Cucumber](wayland/cucumber.json) and [JUnit](wayland/junit.xml)
- [X11 Cucumber](x11/cucumber.json) and [JUnit](x11/junit.xml)
