# Inspect an Extrude distance tolerance

The published rectangular Extrusion1 has a literal 8 mm distance. Its
Distance A menu opens Tolerance using the validated distance. The window
shows Model Value 8.000 mm, the Default type and the twelve recorded
tolerance types. Cancel leaves the active edit at 8 mm and preserves the
feature and document generation.

The pinned source starts from an extrusion created in-session. That path
uses a named dimension and exercises the same tolerance controls. The same
runs cover List Parameters, Select Feature Dimension and opening the
Parameters table through the distance menu.

| Linux backend | Published fixture | Pinned source |
| --- | --- | --- |
| Wayland on COSMIC | [Four scenarios and 36 steps](wayland/published.xml) | [Four scenarios and 36 steps](wayland/pinned.json) |
| X11 under Xvfb and Openbox | [Four scenarios and 36 steps](x11/published.xml) | [Four scenarios and 36 steps](x11/pinned.json) |
