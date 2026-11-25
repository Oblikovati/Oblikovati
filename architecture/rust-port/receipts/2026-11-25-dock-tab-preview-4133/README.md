# Dock tab insertion preview

Detaching the Appearance panel from the left changes the width and center
of the Material dock on the right. The previous drag used the center
measured before detachment and ended outside the center target. The
scenario now measures the rendered Material dock after detachment while
the X11 pointer still holds the Appearance tab. The violet fill and ghost
tab show the insertion preview before release.
[Missed target](missed-target.png) · [Insertion preview](insertion-preview.png)

All ten steps passed with the published Part. After release, the panels
shared one dock and switched contents when the other tab was selected.
The Part stayed visible. The separate native dock-tab regression passed.
[Joined tabs](joined-tabs.png) · [Scenario report](cucumber.json) ·
[Scenario and fixture hashes](current-inputs.sha256)

Verification used Linux X11, Xvfb, and Vulkan Lavapipe. The build used
`d3018e1af` with the changed scenario driver and pending local filesystem
and Ribbon Customization edits.
