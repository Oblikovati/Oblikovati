# Typed drawing sheet lifecycle

The API 0.166.0 file owner serves `drawing.listSheets`,
`drawing.addSheet`, `drawing.setActiveSheet`, and `drawing.removeSheet`.
Listing returns the authored order, the primary model reference, paper
dimensions, border and title block state, and the active flag. New sheets
inherit the standard border and title block. Standard sizes apply paper
orientation and custom sizes use the supplied millimetre dimensions.

The drawing history owns all three edits, including the active sheet because
it is part of the saved drawing recipe. Removing a sheet removes its views
with it. The active index follows the Go reference and is clamped to the last
remaining sheet if needed. A drawing must keep at least one sheet. With
updates deferred, the API can list the frozen recipe. Reopen with updates
enabled before editing it because the saved display cache is keyed to that
recipe.

## Verification

The eight drawing owner tests passed. The sheet lifecycle test checks
standard and custom paper, invalid input, the displayed projection on the
active sheet, last-sheet refusal, undo and redo, and native-format save and
reopen.

```sh
cargo test --locked -p oblikovati-head --lib tests::document_creation_benchmark::drawing
```

The native Wayland desktop, with `DISPLAY` unset, served all four methods
through MCP Bridge. The run created a drawing with two views on an A3
landscape sheet, added an A4 portrait sheet, activated the A3 sheet,
removed it, and checked undo and redo. A 1300 × 750 drawing crop differed
by 350,000 pixels between A3 and A4. Switching back and restoring the A3
sheet reproduced the original crop pixel for pixel. Removing and redoing
the removal reproduced the A4 crop. A fresh native process reopened the
saved drawing and reported the A4 sheet as active.

Rapid close and reopen in one native process triggered the ImGui
`TabBarCalcTabID` docking assertion. A control run reproduced the
assertion using only create, save, close, and open on a default drawing.

## Drawing curve gap

The drawing curve end-to-end reference test requires a model edge key
for each projected box segment.
The current HLR adapter obtains grouped output edge shapes from
`HLRToShape::compound_of_edges`. It retains projected geometry and
classification, while the native presentation snapshot retains sampled
points and visibility. The typed curve route still needs a checked path from
HLR output spans to persistent source edge identities.
