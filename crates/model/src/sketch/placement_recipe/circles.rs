// SPDX-License-Identifier: GPL-2.0-only
use super::{fields::field, *};
use crate::sketch::circle::require_radius_value;
impl SketchPlacementRecipe {
    /// Author one circle with its diameter field; example: `SketchPlacementRecipe::circle([2.,3.],5.)?`.
    pub fn circle(center: [f64; 2], radius: f64) -> Result<Self, SketchError> {
        require_radius_value(radius)?;
        let recipe = Self {
            circular_checks: Vec::new(),
            points: vec![center],
            members: vec![SketchPlacementMember {
                geometry: SketchPlacementGeometry::Circle { center: 0, radius },
                construction: false,
            }],
            // Go aa8297d1 CircleRecipe: the intrinsic circle has three DOF; no extra relations.
            relations: vec![],
            fields: vec![diameter_field(center, radius)],
        };
        recipe.validate()?;
        Ok(recipe)
    }
}
fn diameter_field(center: [f64; 2], radius: f64) -> SketchPlacementField {
    field(
        "Diameter",
        2. * radius,
        [center, [center[0] + radius, center[1]]],
        SketchPlacementBinding::Diameter(0),
    )
}
