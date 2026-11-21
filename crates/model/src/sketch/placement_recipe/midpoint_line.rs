// SPDX-License-Identifier: GPL-2.0-only
//! Describe a line whose midpoint remains an authored sketch point.
use super::{SketchError, SketchPlacementBinding, SketchPlacementRecipe, SketchPlacementRelation};

impl SketchPlacementRecipe {
    /// Describe a midpoint line with full-length and angle fields.
    /// Example: `SketchPlacementRecipe::midpoint_line([0., 0.], [3., 4.])?`.
    pub fn midpoint_line(center: [f64; 2], end: [f64; 2]) -> Result<Self, SketchError> {
        super::require_position(center)?;
        super::require_position(end)?;
        let opposite = std::array::from_fn(|axis| center[axis] - (end[axis] - center[axis]));
        let mut recipe = Self::line_chain(&[opposite, end], false)?;
        recipe.points.push(center);
        recipe
            .relations
            .push(SketchPlacementRelation::Midpoint { point: 2, line: 0 });
        let Some(angle) = recipe.fields.get_mut(1) else {
            return Err(SketchError::Definition(
                "midpoint line fields: expected a length and an angle".into(),
            ));
        };
        angle.binding = SketchPlacementBinding::LineAngle(0);
        recipe.validate()?;
        Ok(recipe)
    }
}
