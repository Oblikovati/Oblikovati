// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn centers(
    [first, last]: [[f64; 2]; 2],
    width: f64,
) -> Result<SketchSlotLayout, SketchError> {
    let (unit, _) = direction(first, last)?;
    let offset = [-unit[1] * (width / 2.0), unit[0] * (width / 2.0)];
    let negative = offset.map(|value| -value);
    Ok(SketchSlotLayout {
        points: vec![
            shifted(first, offset),
            shifted(last, offset),
            shifted(last, negative),
            shifted(first, negative),
            last,
            first,
        ],
        // Go aa8297d1: clockwise caps bulge away from the slot centerline.
        members: [
            SketchPathMember::Line([0, 1]),
            arc_member(4, [1, 2], false),
            SketchPathMember::Line([2, 3]),
            arc_member(5, [3, 0], false),
        ],
    })
}
pub(super) fn overall(
    [first, last]: [[f64; 2]; 2],
    width: f64,
) -> Result<SketchSlotLayout, SketchError> {
    let (unit, length) = direction(first, last)?;
    if length <= width {
        return Err(SketchError::Definition(format!(
            "slot overall length {length}, width {width}: expected length greater than width"
        )));
    }
    let half = unit.map(|value| value * (width / 2.0));
    centers(
        [
            shifted(first, half),
            shifted(last, half.map(|value| -value)),
        ],
        width,
    )
}
pub(super) fn slot_center(
    [center, end]: [[f64; 2]; 2],
    width: f64,
) -> Result<SketchSlotLayout, SketchError> {
    require_position(center)?;
    require_position(end)?;
    centers(
        [[2.0 * center[0] - end[0], 2.0 * center[1] - end[1]], end],
        width,
    )
}
