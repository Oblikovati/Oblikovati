# Point Cloud RGB and Intensity Display Modes

## Context

Attached point clouds currently display as one cyan marker batch. The renderer can already carry
per-vertex colors through `renderer.DrawItem.Colors` and `head/viewport.Flatten`, but attached
clouds do not preserve or route scan color channels:

- `model/pointcloud` stores only XYZ points.
- ASCII XYZ/PTS parsing ignores trailing intensity/RGB columns.
- PLY import reads only vertex positions.
- LAS and E57 readers explicitly ignore intensity and color.
- `app.PointCloudItems` calls `renderer.PointMarkers` with `renderer.PointCloudColor`.

Generic client graphics can display per-vertex colors, but that path is separate from persisted,
attached point-cloud objects.

## Goal

Add an explicit point-cloud display mode that lets users show attached clouds using:

- `default`: existing cyan markers.
- `rgb`: per-point RGB from the scan file when available.
- `intensity`: per-point grayscale mapped from scan intensity when available.

The user must be able to choose the mode from a dropdown on the ribbon near the existing point
cloud tool buttons. Existing documents and clouds without channel data keep the current cyan
appearance.

## Public API

Follow ADR-0018 contract-first order in `../Oblikovati.API`:

1. Add `types.PointCloudDisplayMode` with values `Default`, `RGB`, and `Intensity`.
2. Add `DisplayMode` to `wire.PointCloudInfo`.
3. Add `wire.SetPointCloudDisplayModeArgs`.
4. Add `wire.MethodPointCloudsSetDisplayMode = "pointClouds.setDisplayMode"`.
5. Add `client.PointClouds.SetDisplayMode(name string, mode types.PointCloudDisplayMode)`.

The wire method validates the mode and returns the updated `PointCloudInfo`.

## Model And Import

Introduce a point-cloud sample value in `model/pointcloud` that carries XYZ plus optional RGB and
intensity channels. Keep existing point-only constructors and accessors for geometry consumers.

Readers decode channel data when the source format exposes it:

- ASCII: support `x y z intensity`, `x y z r g b`, and `x y z intensity r g b`.
- PLY: read common vertex properties `red`, `green`, `blue`, and `intensity` when present.
- LAS: read intensity for all point record formats and RGB for formats that include color.
- E57: preserve the current XYZ-only behavior in this pass. E57 clouds can select `rgb` or
  `intensity`, but they render with cyan fallback until a later E57 channel-decoding extension.

Unit scaling applies only to XYZ. RGB and intensity are value channels and are not scaled.

## Rendering

Add a renderer helper that builds point-marker crosses with optional per-point colors. Because each
displayed scan point expands to six marker vertices, the helper repeats the point color across those
six vertices.

For `rgb`, use scan RGB when present. For missing RGB data, fall back to `PointCloudColor`.

For `intensity`, normalize over the cloud's decoded intensity range and produce grayscale RGBA. If
the cloud has no intensity values or the range is degenerate, fall back to `PointCloudColor`.

The display cache must keep points and their channels aligned through budget sampling, crop
filtering, LOD thinning, and frustum clipping.

## Persistence

Do not duplicate RGB/intensity channels in point-cloud records. The `.obk` resource table already
stores the original scan bytes; on open, records re-decode from the embedded resource. Add the
selected display mode to point-cloud records so the user's mode choice round-trips.

Older documents without a display mode load as `default`.

## Ribbon UI

Add a point-cloud display-mode dropdown to the existing Point Cloud ribbon panel, adjacent to the
other point-cloud tool buttons. The dropdown shows `Default`, `RGB`, and `Intensity`.

Behavior:

- Disabled unless one point cloud is selected.
- Reflects the selected cloud's current mode.
- Selecting an option updates the selected cloud through the app/session command path.
- The viewport updates immediately.
- If the requested data is absent, the cloud still switches mode but renders with cyan fallback; no
  destructive conversion occurs.

## Error Handling

Invalid API mode values return an error naming the offending mode and the expected values.

Malformed scan records continue to follow the existing warn-and-continue import policy. Unsupported
or absent color/intensity channels are not import errors.

## Testing

Add focused tests for:

- Verification that the previous default render path remains cyan.
- ASCII parsing of intensity, RGB, and intensity+RGB columns.
- PLY parsing of RGB/intensity vertex properties.
- LAS minimal records with intensity and RGB point formats.
- Point-cloud display cache alignment between sampled points and channel colors.
- Renderer marker helper repeating one per-point color over six line vertices.
- Router/client `pointClouds.setDisplayMode`.
- Persistence round-trip of display mode.
- Ribbon registry/UI state showing and enabling the dropdown near point-cloud tools.

## Non-Goals

- Native GPU point sprites.
- Color legends or editable intensity ramps.
- E57 multi-scan support.
- Applying scan RGB/intensity to generic client graphics.
