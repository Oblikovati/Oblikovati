// SPDX-License-Identifier: GPL-2.0-only
use super::{
    fields::{distance, field},
    *,
};
use crate::sketch::circle::require_radius_value;
impl SketchPlacementRecipe {
    /// Keep arc endpoints and intrinsic circularity; example: `SketchPlacementRecipe::arc([0.,0.],[[2.,0.],[0.,2.]],true)?`.
    pub fn arc(center: [f64; 2], endpoints: [[f64; 2]; 2], ccw: bool) -> Result<Self, SketchError> {
        require_arc_points(center, endpoints)?;
        let [start, end] = endpoints;
        let recipe = Self {
            circular_checks: Vec::new(),
            points: vec![center, start, end],
            members: vec![arc_member(ccw)],
            // Go aa8297d1 ArcRecipe: circularity is structural; no extra equation is authored.
            relations: vec![],
            fields: arc_fields(center, endpoints),
        };
        recipe.validate()?;
        Ok(recipe)
    }
}
fn arc_fields(center: [f64; 2], [start, end]: [[f64; 2]; 2]) -> Vec<SketchPlacementField> {
    vec![
        field(
            "Radius",
            distance([center, start]),
            [center, start],
            SketchPlacementBinding::Radius(0),
        ),
        super::fields::sweep_field(center, [start, end]),
    ]
}
fn require_arc_points(center: [f64; 2], endpoints: [[f64; 2]; 2]) -> Result<(), SketchError> {
    let [start, end] = endpoints;
    if start == end {
        return Err(SketchError::Definition(format!(
            "arc endpoints {endpoints:?}: expected distinct start and end"
        )));
    }
    require_radius_value(distance([center, start]))?;
    require_radius_value(distance([center, end]))?;
    Ok(())
}
fn arc_member(ccw: bool) -> SketchPlacementMember {
    SketchPlacementMember {
        geometry: SketchPlacementGeometry::Path(SketchPathMember::Arc {
            center: 0,
            endpoints: [1, 2],
            counter_clockwise: ccw,
        }),
        construction: false,
    }
}
