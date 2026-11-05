// SPDX-License-Identifier: GPL-2.0-only
use super::{fields::field, *};
use crate::sketch::ellipse::{normalized_ellipse_axis, require_ellipse_frame};
impl SketchPlacementRecipe {
    /// Preserve an ellipse frame and placement fields; example: `SketchPlacementRecipe::ellipse([0.,0.],[1.,0.],[5.,2.])?`.
    pub fn ellipse(center: [f64; 2], axis: [f64; 2], radii: [f64; 2]) -> Result<Self, SketchError> {
        require_ellipse_frame(axis, radii)?;
        let major_axis = normalized_ellipse_axis(axis)?;
        let recipe = Self {
            points: vec![center],
            members: vec![ellipse_member(major_axis, radii)],
            relations: vec![],
            circular_checks: vec![],
            fields: ellipse_fields(center, major_axis, radii),
        };
        recipe.validate()?;
        Ok(recipe)
    }
}
fn ellipse_fields(center: [f64; 2], axis: [f64; 2], radii: [f64; 2]) -> Vec<SketchPlacementField> {
    let minor = [-axis[1], axis[0]];
    vec![
        field(
            "Major",
            radii[0],
            radial_witness(center, axis, radii[0]),
            SketchPlacementBinding::EllipseRadius(0),
        ),
        // Go aa8297d1 EllipseRecipe: the minor input steers placement without a live dimension.
        field(
            "Minor",
            radii[1],
            radial_witness(center, minor, radii[1]),
            SketchPlacementBinding::SteeringLength,
        ),
    ]
}
fn radial_witness(center: [f64; 2], axis: [f64; 2], radius: f64) -> [[f64; 2]; 2] {
    [
        center,
        [center[0] + axis[0] * radius, center[1] + axis[1] * radius],
    ]
}

fn ellipse_member(major_axis: [f64; 2], radii: [f64; 2]) -> SketchPlacementMember {
    SketchPlacementMember {
        geometry: SketchPlacementGeometry::Ellipse {
            center: 0,
            major_axis,
            radii,
        },
        construction: false,
    }
}
