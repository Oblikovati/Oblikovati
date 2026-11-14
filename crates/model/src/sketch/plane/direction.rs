// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, require_finite, vector_length};

pub(crate) fn normalize_direction(direction: [f64; 3]) -> Result<[f64; 3], SketchError> {
    require_finite(&direction, "orientation direction")?;
    let scale = direction
        .iter()
        .map(|value| value.abs())
        .fold(0.0, f64::max);
    if scale == 0.0 {
        return Err(SketchError::Definition(format!(
            "sketch orientation direction {direction:?}: expected a finite nonzero direction"
        )));
    }
    // A direction has no length unit; scaling first prevents overflow and subnormal loss (#3736).
    let scaled = direction.map(|value| value / scale);
    let length = vector_length(scaled);
    Ok(scaled.map(|value| value / length))
}

pub(crate) fn point_difference(first: [f64; 3], second: [f64; 3]) -> [f64; 3] {
    let difference = std::array::from_fn(|component| second[component] - first[component]);
    if difference.iter().all(|value| value.is_finite()) {
        return difference;
    }
    overflowing_point_difference(first, second, difference)
}
fn overflowing_point_difference(
    first: [f64; 3],
    second: [f64; 3],
    difference: [f64; 3],
) -> [f64; 3] {
    // Normalize before subtraction only when the finite-point difference overflows (#3740).
    let scale = first
        .into_iter()
        .chain(second)
        .map(f64::abs)
        .fold(0.0, f64::max);
    std::array::from_fn(|component| {
        scaled_difference(
            first[component],
            second[component],
            difference[component],
            scale,
        )
    })
}

fn scaled_difference(first: f64, second: f64, difference: f64, scale: f64) -> f64 {
    if difference.is_finite() {
        return difference / scale;
    }
    second / scale - first / scale
}
