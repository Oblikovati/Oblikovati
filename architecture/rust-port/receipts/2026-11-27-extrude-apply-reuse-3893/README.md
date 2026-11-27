# Reuse an Extrude sketch after Apply

With **Keep sketches visible on (+)** checked, Apply and Create New
commits the rectangular extrusion and keeps the two-profile sketch
visible in one undo step. The second draft waits for a profile. A
viewport click outside the sketch regions leaves it waiting. A click
inside the circle starts a circular preview. The document generation,
undo history, and committed rectangular feature and body stay the same
through both clicks. The native regression uses the bundled copy of the
published sketch fixture.
[Published input checksums](published-inputs.sha256).

Both desktops passed the shared five-step case, the eight-step fixture
variation, the six-step settings carry-over case, and the native
regression.

| Backend | Shared Properties | Two profiles | Carried settings | Native regression |
| --- | --- | --- | --- | --- |
| Wayland | [5 passed](wayland-shared.xml) | [8 passed](wayland-two-profile.xml) | [6 passed](wayland-carry-settings.xml) | [1 passed](wayland-native.txt) |
| X11 | [5 passed](x11-shared.xml) | [8 passed](x11-two-profile.xml) | [6 passed](x11-carry-settings.xml) | [1 passed](x11-native.txt) |
