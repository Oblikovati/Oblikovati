# Float and redock the Material Browser

The previous scenario expected exact original viewport bounds after
redocking. Both edges moved 9 px when the browser redocked. The published
case asks the viewport to adjust to the resulting workspace. The updated
check verifies that Material actually docks at the right edge and that the
viewport fills the gap between the rendered Model and Material docks,
allowing for the splitters. The same Part must remain active and visibly
shaded.

All eight steps passed under Linux X11, Openbox, Xvfb, and Vulkan Lavapipe
with the published Part. The captures show the browser floating and the
adjusted viewport after redocking.
[Floating](floating.png) · [Redocked](redocked.png) ·
[Scenario report](cucumber.json) · [Input hashes](current-inputs.sha256)

The build used revision `7b6596aac` with the updated checks and pending
local filesystem and Ribbon Customization edits.
