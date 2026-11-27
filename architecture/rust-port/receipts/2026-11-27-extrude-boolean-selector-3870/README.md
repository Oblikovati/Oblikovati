# Switch the Extrude Boolean selector style

The published scenario uses the circular sketch on an existing solid.
The native test counts four operation icons, selects Join, and switches
the Operation selector to Dropdown. It opens the named options and
checks that Join, Cut, Intersect, and New Solid are enabled. The
selected Join and the boss preview survive the presentation change.

The four controls use the desktop catalog's original Boolean SVGs.

[Wayland JUnit](wayland/junit.xml) and [X11 JUnit](x11/junit.xml)
record the published scenario. The pinned scenario revision predates it.
