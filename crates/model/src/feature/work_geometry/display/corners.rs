// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn validate(corners: [[f64; 3]; 2]) -> Result<(), WorkGeometryError> {
    if corners.iter().flatten().all(|value| value.is_finite()) {
        return Ok(());
    }
    Err(WorkGeometryError::DisplayCorners(corners))
}
pub(super) fn derived(
    frame: SketchPlane,
    units: &UnitsOfMeasure,
) -> Result<[[f64; 3]; 2], WorkGeometryError> {
    // Go #1851 uses a five-centimetre half-extent; display preferences must not rescale it.
    const HALF_EXTENT_CM: f64 = 5.0;
    let recipe = frame.recipe();
    let origin = recipe.origin.map(|value| value * units.working_scale());
    let diagonal: [f64; 3] =
        std::array::from_fn(|axis| HALF_EXTENT_CM * (recipe.x_axis[axis] + recipe.y_axis[axis]));
    let corners = [
        std::array::from_fn(|axis| origin[axis] - diagonal[axis]),
        std::array::from_fn(|axis| origin[axis] + diagonal[axis]),
    ];
    validate(corners)?;
    Ok(corners)
}
pub(super) fn same_recipe(left: Option<[[f64; 3]; 2]>, right: Option<[[f64; 3]; 2]>) -> bool {
    match (left, right) {
        (None, None) => true,
        (Some(left), Some(right)) => left
            .iter()
            .flatten()
            .zip(right.iter().flatten())
            .all(|(left, right)| left.to_bits() == right.to_bits()),
        _ => false,
    }
}
