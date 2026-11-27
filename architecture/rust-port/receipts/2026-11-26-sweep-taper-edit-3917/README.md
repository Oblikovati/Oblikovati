# Sweep taper edit

The published `sweep-03` journey creates Sweep1 at 5 degrees, reopens its
Properties from the Model Browser, and checks the retained inputs and the
angle. It compares the tapered and straight preview widths and checks that
accepting zero leaves Sweep1 with one solid and a saved zero taper.

- [Wayland Cucumber](wayland/cucumber.json) and [JUnit](wayland/junit.xml)
- [X11 Cucumber](x11/cucumber.json) and [JUnit](x11/junit.xml)

The Linux build used Martensite v0.14.2. Wayland ran on COSMIC. X11 ran
under Xvfb with Lavapipe. The [input hashes](current-inputs.sha256) identify
the published scenario and the source file copied for each run.
