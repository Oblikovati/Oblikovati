# Cancel populated drawing layout creation

The two cancellation scenarios each passed 16 steps on Wayland and X11.
The direct cancellation left the original Part active without adding a
drawing tab. The nested cancellation restored the exact proposed component
name before the outer dialog closed.

The revised assertions recognize the current light paper in both preview
regions and allow adjacent field glyphs to appear in a single diagnostic
run. They still check the exact component name, the active document, and the
document-tab count.

All four published scenarios were checked on Wayland. The two confirmation
cases stopped at unbound steps before confirming a component. The
component dialog still disables OK under #3899.

The runs used `5593d2793` with this test change and local filesystem and
sketch-marking test edits. Wayland used headless Weston and frame-loop
input. X11 used Xvfb and Openbox. Both used Vulkan Lavapipe. The input
hashes include the current published OpenPart, whose bytes differ from
the earlier receipt.
[Wayland report](wayland-cucumber.json) · [X11 report](x11-cucumber.json) · [Input hashes](inputs.sha256) · [Wayland previews](template-previews.png)
