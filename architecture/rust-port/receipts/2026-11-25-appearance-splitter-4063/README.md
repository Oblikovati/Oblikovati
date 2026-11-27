# Appearance Browser splitter and the viewport width cap

On the base build, the native pointer returned to the original splitter
position. After release, the viewport's horizontal bounds moved from
`248..1041` to `225..1018`, and Appearance widened from 397 to 420 px.
The width cap reapplied preferred widths every frame and counted the
difference between the capped width and the pointer's position as another
change. The updated layout saves the chosen width and gives the last
resized dock priority when the viewport width cap applies.

All 13 steps of the published Appearance Browser resize case passed under
X11, Openbox, and Vulkan Lavapipe. This includes restoring both viewport
boundaries and the Appearance width within one pixel. A separate native
test widened the dock at the cap, restarted the app, reopened the same
Part, checked identical bounds, and dragged the splitter back.
[Published result](appearance/cucumber.json) ·
[Widened](appearance-splitter-widened-at-cap.png) ·
[Dragged back](appearance-splitter-dragged-back.png)

All 74 filtered GUI workspace tests passed. The native asymmetric-dock
resize regression also passed at 940 and 780 px and after widening.
The new persistence field is optional in existing workspace profiles.

The broader run passed seven of eleven scenarios, up from five. Both
splitter resize scenarios passed.
Three separate cases still failed in both builds under Openbox: closing
Appearance from a tabbed dock, detecting a tab insertion preview, and
recovering the original bounds after floating and redocking a panel.
The two-process restart case needs the dedicated 1440 × 900 X11 display.
[Base suite](docking-suite-base.json) · [Current suite](docking-suite-current.json)

At the extreme end of an active drag, the native splitter can still
temporarily reduce viewport width below the reserved fraction. The final
width is capped on release. The builds used base revision `4f4572c18`
with pending local filesystem and Ribbon Customization edits.
