# Responsive Part and Home workspaces

The audit's 780 × 580 Part and 1100 × 760 Home sizes were reproduced on a
private Linux X11 desktop with Openbox and Vulkan Lavapipe. The saved Part
came from the published `docking-visible-solid-current.opd` fixture. The
release build was captured after resizing and using the real keyboard
shortcut to switch between Home and the Part.

- At 780 × 580 the Model ribbon groups have icons and captions. The 3D
  model, ViewCube, and active document tab are visible together.
  [Narrow Part](part-narrow.png)
- Home uses the width released by the Model and asset docks. Its rail
  shows New Part, New, and Open. The same Part remains available in the
  document strip. [Home](home-medium.png)
- Returning to the Part restores its ribbon, dock widths, and view.
  [Returned Part](part-restored.png)

GUI tests resize a workspace with both side docks from 1440 to 1100 and
780 and back, verify the active long tab and close control at narrow width,
and check that available filenames keep their space as metadata columns
leave the Home list. The 780 px ribbon test opens the last 3D Model group
without scrolling.

The published “Visit Home without leaving an active sketch” scenario passed
all seven steps. The viewport sample follows the live dock bounds and
avoids the sketch axes. The six-step startup observation in this run
included the empty browsers. The later startup fix suspends them.
[Sketch scenario](cucumber.json) · [Startup scenario](startup/cucumber.json)
· [Current startup](../2026-11-25-design-gap-20-startup-home-4063/README.md)

The native Recent test verified both sort directions through the Size
heading with Home in front of the open documents and the asset docks hidden.

The direct native minimum-window test passed resize, viewport preservation,
widening, and close. The separate Main Window Frame BDD scenario reached
its minimum size, then failed at the maximize check in this private desktop.
