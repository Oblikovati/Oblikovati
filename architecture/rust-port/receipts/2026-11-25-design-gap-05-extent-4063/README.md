# Extrude extent dropdowns

From and Termination use native dropdowns with chevrons and show their
current choice. The reference selectors share the 40/60 properties
alignment. Choosing From / To requires both references. Returning to
Sketch Plane restores the numeric preview.

Two published scenarios passed all 14 steps with disposable copies of
the named revolve fixture. The native tests separately checked a
return to numeric preview and an actual To Face pick, preview, and
feature commit. The tests ran on Linux X11 with Openbox and Vulkan
Lavapipe.
[From dropdown](extent-choose-From-To.png) · [Termination dropdown](extent-choose-ToFace.png) ·
[From scenario](scenario-10/cucumber.json) · [Termination scenario](scenario-11/cucumber.json)

The exported extrusion scenario source has SHA-256
`e669b2d674a351779faa012e1ec023576ec61f9431b5f3d8254b3f9434dd272a`.
