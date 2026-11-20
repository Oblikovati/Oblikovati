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
            picked_points: Default::default(),
            circular_checks: Vec::new(),
            points: vec![center, start, end],
            members: vec![arc_member(ccw)],
            // Go aa8297d1 ArcRecipe: circularity is structural; no extra equation is authored.
            relations: vec![],
            fields: arc_fields(center, endpoints, ccw),
        };
        recipe.validate()?;
        Ok(recipe)
    }
}
fn arc_fields(
    center: [f64; 2],
    [start, end]: [[f64; 2]; 2],
    ccw: bool,
) -> Vec<SketchPlacementField> {
    vec![
        field(
            "Radius",
            distance([center, start]),
            [center, start],
            SketchPlacementBinding::Radius(0),
        ),
        arc_sweep_field(center, [start, end], ccw),
    ]
}
/// The arc's unsigned turn, as a dimension can hold it.
fn arc_sweep_field(
    center: [f64; 2],
    [start, end]: [[f64; 2]; 2],
    ccw: bool,
) -> SketchPlacementField {
    let from = (start[1] - center[1]).atan2(start[0] - center[0]);
    let to = (end[1] - center[1]).atan2(end[0] - center[0]);
    let turn = if ccw { to - from } else { from - to };
    field(
        "Sweep",
        turn.rem_euclid(std::f64::consts::TAU),
        [start, end],
        SketchPlacementBinding::ArcSweep(0),
    )
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
        centerline: false,
        geometry: SketchPlacementGeometry::Path(SketchPathMember::Arc {
            center: 0,
            endpoints: [1, 2],
            counter_clockwise: ccw,
        }),
        construction: false,
    }
}
