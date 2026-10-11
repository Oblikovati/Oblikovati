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
