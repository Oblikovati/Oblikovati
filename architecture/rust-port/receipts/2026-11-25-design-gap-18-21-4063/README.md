# Floating properties and Light theme contrast

The native resize check opened the published saved Part, resized the
client from 1440 × 900 to 980 × 540, and widened it again. It compared
each property's actual window bounds with the frame's dock-adjusted
viewport. Extrude shrank its scrollable body while retaining its footer.
Chamfer and Extrude both kept OK and Cancel inside the panel. The test
also clicked Cancel at the short size and confirmed that each draft closed.
[Extrude at 980 × 540](Extrude-980x540.png) ·
[Chamfer at 980 × 540](Chamfer-980x540.png)

Hole, Shell, Move Bodies, and Offset Plane use the same floating layout.
The geometry also recognizes saved dock identities so the dock node can
keep control of its windows.

The Light theme uses slate ink on the disabled OK button and dark ink
for read-only RGBA values. The Theme Editor puts the label before the
duplicate-name field and shows swatch borders and a floating shadow.
The native check inspects the rendered colors and clicks the disabled
action to verify it remains inert. The editor integration test exercises
editing, saving, duplication, and the read-only controls.
[Disabled Chamfer action](light-disabled-chamfer.png) ·
[Light Theme Editor](light-theme-editor.png)
