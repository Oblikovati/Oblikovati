# Point Cloud Render Density Slider

## Context

Attached point clouds now render through a retained native point buffer. The model already supports
per-cloud `MaximumPointCount`, but that budget is a hard count, uses even striding, and belongs to
the cloud metadata. The new control is a viewport performance knob: it should reduce how many scan
points the engine uploads and draws without changing the point cloud data or geometry consumers.

## Goal

Add a point-cloud render density slider where `100%` renders every eligible display point and lower
values render a stable random subset. The subset must be randomly distributed, deterministic across
frames, and refreshed when the density changes.

## Behavior

- The setting is session-level and defaults to `100%`.
- The accepted range is `0..100`; values outside that range are clamped.
- `100%` keeps current behavior.
- `0%` renders no point-cloud points.
- Intermediate values keep approximately that percent of displayed samples.
- Density is render-only. It does not affect crop volumes, snap targets, fitting, work-point
  creation, persistence, source scan resources, or per-cloud `MaximumPointCount`.

## Architecture

Store the density on `app.Session` with explicit getter and setter methods. Expose it in the ribbon
model as a dedicated slider panel on the Surfaces & Mesh tab near the point-cloud controls. The head
renders that panel with the existing Dear ImGui percentage slider wrapper and writes changes back to
the session.

Point filtering belongs in the app render assembly layer, before GPU vertices are built. A stable
hash threshold over cloud identity and point position decides whether each sample survives. This
gives a random-looking distribution without per-frame RNG state or flicker.

Include the density value in `PointCloudDisplayKey()` so a slider change invalidates the retained
point buffer exactly once. The native renderer still skips upload while camera movement is the only
change.

## Testing

Add focused tests for:

- Session density default and clamping.
- Stable random density filtering and approximate retained counts.
- `PointCloudGPUVertices` applying the render density.
- `PointCloudDisplayKey` changing when density changes.
- The Surfaces & Mesh ribbon exposing the render-density slider.

## Non-Goals

- Persisting the render-density setting.
- Per-cloud density controls.
- GPU shader discard.
- Replacing `MaximumPointCount`.
