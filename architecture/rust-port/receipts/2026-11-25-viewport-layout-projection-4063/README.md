# Viewport projection follows dock and window resizing

After widening the right dock, the original desktop projected the camera
target at `[720, 449]`. The live document area's center was about
`[679, 499]`. The updated viewport uses the latest document rectangle
for projection and presents the refreshed annotation coordinates with it.
The native regression changes the dock width, shrinks the client to
940 × 680, and checks that a panned and zoomed camera keeps its settings.
Previous View restores the recorded camera in the current rectangle.
[Narrow Part](narrow-part.png)

All ten published Orbit cases and 94 steps passed. All seven published
Zoom cases and 74 steps passed. The native asymmetric-dock regression
and the viewport library suite passed, with 186 library tests run
successfully and one ignored. An exploratory Entity Selection run passed
the four solid and empty-space cases. Its four 3D Sketch cases still have
undefined steps. [Orbit](orbit/cucumber.json) · [Zoom](zoom/cucumber.json) ·
[Scenario and primary Part hashes](current-inputs.sha256)

All six steps of the published dock restart scenario passed on the rebuilt
desktop. The image check used the actual saved document rectangle to
locate the same fixture face and background. The two process captures
were byte-for-byte identical. [Restart scenario](restart/cucumber.json) ·
[Restart capture](restart/part.png) · [Measurements](restart/restart-summary.json)

Verification used Linux X11 with Xvfb and Vulkan Lavapipe. The builds used
revision `ab5a0be84` with the projection changes and pending local
filesystem and Ribbon Customization edits.
