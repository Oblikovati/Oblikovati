// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn arc(
    center: [f64; 2],
    [start, end]: [[f64; 2]; 2],
    width: f64,
    ccw: bool,
) -> Result<SketchSlotLayout, SketchError> {
    let (from, radius) = direction(center, start)?;
    let (to, _) = direction(center, end)?;
    let half = width / 2.0;
    if radius - half <= 0.0 {
        return Err(SketchError::Definition(format!(
            "arc slot radius {radius}, width {width}: expected a positive inner radius"
        )));
    }
    let outer = radius + half;
    let inner = radius - half;
    Ok(SketchSlotLayout {
        points: vec![
            shifted(center, from.map(|v| v * outer)),
            shifted(center, from.map(|v| v * inner)),
            shifted(center, to.map(|v| v * outer)),
            shifted(center, to.map(|v| v * inner)),
            center,
            center,
            end,
            start,
        ],
        // Preserve the Go boundary order and separate concentric-center ownership.
        members: [
            arc_member(4, [0, 2], ccw),
            arc_member(6, [2, 3], ccw),
            arc_member(5, [3, 1], !ccw),
            arc_member(7, [1, 0], ccw),
        ],
    })
}
pub(super) fn center_point(
    center: [f64; 2],
    start: [f64; 2],
    sweep: f64,
    width: f64,
) -> Result<SketchSlotLayout, SketchError> {
    if !sweep.is_finite() {
        return Err(SketchError::Definition(format!(
            "arc slot sweep {sweep}: expected a finite angle"
        )));
    }
    require_position(center)?;
    require_position(start)?;
    let delta = [start[0] - center[0], start[1] - center[1]];
    let (sin, cos) = sweep.sin_cos();
    let end = shifted(
        center,
        [
            delta[0] * cos - delta[1] * sin,
            delta[0] * sin + delta[1] * cos,
        ],
    );
    arc(center, [start, end], width, sweep > 0.0)
}
