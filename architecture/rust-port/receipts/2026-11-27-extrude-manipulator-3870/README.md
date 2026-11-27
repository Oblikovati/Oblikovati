# Synchronize Distance A with its viewport manipulator

The published scenario opens a 10 mm numerical preview of a rectangle.
The test grabs the handle in the viewport and moves it about half the
original length along the projected extrusion axis. It checks the
resulting depth of about 15 mm against the rendered dimension and Distance A
field. Entering 8 mm in the panel moves the dimension and the submitted
geometry back to 8 mm. OK remains enabled and the preview leaves the
document without a committed feature.

[Wayland JUnit](wayland/junit.xml) and [X11 JUnit](x11/junit.xml)
record the published scenario. The pinned scenario revision predates
this viewport contract.
