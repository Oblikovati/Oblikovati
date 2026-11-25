# File menu footer and active controls

The native File menu ran on Linux under Xvfb, Openbox, and Vulkan Lavapipe.
The dark theme captures show the open File tab in brand violet, an accent
underline on both current Recent values, aligned 16 px footer icons, and
the shadow at the popup perimeter.
[Part capture](file-menu-part.png) · [Home capture](file-menu-home.png)

Three focused native checks passed. They measured the 32 px command and
footer pitch, switched both Recent controls, opened Application Options,
and checked dismissal and contextual commands.

All eight published File menu scenarios passed, covering 67 steps across
Part, Drawing and Home, including dismissal and the New and Save As menus.
[Cucumber](cucumber.json) · [JUnit](junit.xml) ·
[Scenario arguments](scenario-arguments.json)

The scenario source was the exported
`features/application/file-menu/file-menu.feature`, with SHA-256
`2c5cd3650dc5a2e2aa77e5378dae69a4ab29c2786c1fb3f261934f67bb6e212d`.
[Source report](source-revision.json)
