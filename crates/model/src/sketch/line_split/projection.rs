// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchEntityId, SketchError};
use crate::sketch::geometry::require_position;
pub(super) fn split_position(
    line: SketchEntityId,
    endpoints: [[f64; 2]; 2],
    pick: [f64; 2],
) -> Result<[f64; 2], SketchError> {
    require_position(pick)?;
    let [a, b] = endpoints;
    let delta = [b[0] - a[0], b[1] - a[1]];
    require_position(delta)?;
    let parameter = projected_fraction(line, a, delta, pick)?;
    let position = [a[0] + parameter * delta[0], a[1] + parameter * delta[1]];
    require_position(position)?;
    if position == a || position == b {
        return Err(split_projection_error(line, parameter));
    }
    Ok(position)
}
fn projected_fraction(
    line: SketchEntityId,
    start: [f64; 2],
    delta: [f64; 2],
    pick: [f64; 2],
) -> Result<f64, SketchError> {
    let scale = delta[0].abs().max(delta[1].abs());
    if scale == 0.0 {
        return Err(split_projection_error(line, f64::NAN));
    }
    let direction = delta.map(|value| value / scale);
    let offset = [(pick[0] - start[0]) / scale, (pick[1] - start[1]) / scale];
    require_position(offset)?;
    let parameter = (offset[0] * direction[0] + offset[1] * direction[1])
        / (direction[0] * direction[0] + direction[1] * direction[1]);
    if !parameter.is_finite() || parameter <= 0.0 || parameter >= 1.0 {
        return Err(split_projection_error(line, parameter));
    }
    Ok(parameter)
}
fn split_projection_error(line: SketchEntityId, parameter: f64) -> SketchError {
    SketchError::Definition(format!(
        "split line {line:?}, projected parameter {parameter:?}: expected a representable point strictly inside a nonzero finite segment"
    ))
}
