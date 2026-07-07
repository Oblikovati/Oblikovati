# Point Cloud Display Controls

## Context

Point clouds have a combined ribbon panel for display mode, render density, and the
intensity color ramp. The current explicit `Default` display mode falls back to the
renderer's cyan point-cloud color, which reads as a palette choice rather than a neutral
scan view. The retained GPU point path also accepts a fixed point size from the head, so
users cannot tune point thickness for dense or sparse scans.

## Goal

Improve the point-cloud display panel without changing import defaults. Newly imported
point clouds continue to start in RGB mode. When a user selects the explicit `Default`
mode, points render as neutral grey. The panel also gains a point-size slider below the
density slider and shows a CloudCompare-style intensity distribution between the low and
high intensity color swatches.

## Behavior

- New point clouds remain in `types.PointCloudDisplayModeRGB`.
- `types.PointCloudDisplayModeDefault` renders with a greyscale point color instead of
  the existing cyan fallback.
- Point size is a session-level viewport setting.
- Point size defaults to `1` and clamps to the inclusive range `1..10`.
- The Point Cloud Display ribbon panel stacks controls in this order: display-mode
  selector, density slider, point-size slider, then intensity controls when applicable.
- Intensity controls appear only when the selected cloud is in intensity mode.
- The intensity controls place the low swatch at the left, the high swatch at the right,
  and an area chart between them showing the selected cloud's intensity distribution.
- If a selected intensity cloud has no valid intensity spread, the chart renders as an
  empty baseline while the swatches remain editable.

## Architecture

Keep the state and ribbon data in `app` and keep `head/ui` as a thin renderer of that
model. Add explicit `PointCloudPointSize()` and `SetPointCloudPointSize()` methods to
`app.Session`, initialized in `newSession`. The retained GPU upload path reads the session
point size rather than a hardcoded value, and point-size changes invalidate or update the
native point buffer so the visible result changes immediately.

Extend the existing `RibbonPanel` model rather than drawing special controls ad hoc in the
head. The panel can carry a render-density slider, a point-size slider, and optional
intensity-ramp data. The head renders multiple sliders vertically under the selector using
the same slider wrapper style as density.

Build the intensity chart data from the selected cloud's displayed samples so crops and
display budget are reflected in the visible distribution. Use fixed histogram bins for a
small, deterministic ribbon payload. The chart is informational only; it does not remap or
clip intensity values.

## Data Flow

`BuildRibbon` injects display controls into the Point Cloud Display panel. It reads
session density, session point size, selected-cloud display mode, and selected-cloud
histogram data. `head/ui/chrome_ribbon.go` draws those controls and writes edits back to
the session.

Rendering uses the selected cloud's display mode as before. RGB mode uses decoded color
when present. Intensity mode interpolates between the session low/high ramp colors.
Default mode and missing-channel fallback use the neutral grey point-cloud color.

## Error Handling

Point size setter clamps invalid values into `1..10`; no user-visible error is needed for
a live slider. Histogram generation tolerates missing selection, missing intensity data,
and collapsed intensity ranges by returning no bars. Color setters continue to force
opaque alpha and clamp color channels through the existing ramp path.

## Testing

Add focused tests for:

- Point-size default and clamp behavior.
- The point-size value being exposed in the ribbon below render density.
- The retained point-cloud display key or upload path responding to point-size changes.
- Default display mode using neutral grey while imported clouds still default to RGB.
- Intensity histogram generation for populated, empty, and collapsed ranges.
- The intensity panel model exposing separated swatches with chart data only in intensity
  mode.

## Non-Goals

- Persisting point size.
- Per-cloud point size.
- Changing RGB as the import default.
- Making the intensity histogram interactive.
- Adding public API or wire methods for these controls.
