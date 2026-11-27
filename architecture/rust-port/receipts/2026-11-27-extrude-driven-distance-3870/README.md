# Drive Extrude with a part parameter

The published scenario opens a part with a rectangular profile and a
4 mm user parameter named `Depth`. Entering `Depth * 2` in Distance A
enables OK and produces an 8 mm preview. After acceptance, changing
`Depth` to 5 mm through Manage → Parameters rebuilds the same extrusion
to 10 mm in the viewport. The feature-owned distance retains the source
expression, and reopening its editor previews the new depth.

The scenario also exercises closing the Parameters panel with an edited
equation active. The panel receives the click without starting a model
pick against the previous geometry generation.

[Wayland JUnit](wayland/junit.xml) and [X11 JUnit](x11/junit.xml)
record the published scenario. The pinned scenario revision predates
this expression contract.
