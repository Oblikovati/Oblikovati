# Extrude default distance and numeric input styling

New Extrude drafts start at 10.00 mm. Selecting the value allows it to be
replaced directly. Apply & Create New starts another draft at the default
distance and waits for a new sketch choice. Chamfer and Fillet draw an
outline around the numeric field.

The Extrude direction buttons have equal widths in the 280 px panel.
The active appearance and direction labels are retained. At higher
text scales or reduced width, the icons keep their individual tooltips.

Six GUI integration tests passed. Native checks covered the 10.00 mm
seed after both free and constrained sketch creation, replacing it
with 8 mm, toggling Preview, and opening a fresh draft after Apply.
They also measured the direction icon pitch and checked that OK and
Cancel stay visible at a native window size of 1100 × 390.
[Extrude](extrude-properties.png) · [Chamfer](chamfer-properties.png) ·
[Short viewport](extrude-short-properties.png) · [Preview and Apply](extrude-profile-and-actions.png)

A later X11 check verified the fitted Chamfer panel and measured the
remaining space below Cancel at under 20 px.
[Fitted Chamfer](chamfer-fitted.png)
