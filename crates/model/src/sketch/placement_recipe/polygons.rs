// SPDX-License-Identifier: GPL-2.0-only
use super::{fields::field, geometry::chain_members, *};
use crate::sketch::SketchPolygonSeed;
impl SketchPlacementRecipe {
    /// Preserve polygon regularity for preview and commit; example: `SketchPlacementRecipe::polygon(seed)?`.
    pub fn polygon(seed: SketchPolygonSeed) -> Result<Self, SketchError> {
        let (mut points, radius) = seed.boundary()?;
        let fields = polygon_fields(seed.center, points[0], radius, seed.sides);
        points.push(seed.center);
        let recipe = Self {
            points,
            members: polygon_members(seed.sides, radius),
            fields,
            relations: SketchPlacementRelation::polygon(seed.sides).collect(),
            circular_checks: vec![],
        };
        recipe.validate()?;
        Ok(recipe)
    }
}
impl SketchPlacementRelation {
    pub(crate) fn polygon(sides: usize) -> impl Iterator<Item = Self> {
        let radial = (0..sides).map(move |point| Self::PointOnCircle {
            point,
            circle: sides,
        });
        // Go aa8297d1 (#2014): n-1 equal chord relations remove the remaining shape freedoms.
        let equal = (0..sides.saturating_sub(1)).map(|i| Self::EqualLength([i, i + 1]));
        radial.chain(equal)
    }
}
fn polygon_fields(
    center: [f64; 2],
    vertex: [f64; 2],
    radius: f64,
    sides: usize,
) -> Vec<SketchPlacementField> {
    let vector = [vertex[0] - center[0], vertex[1] - center[1]];
    vec![
        field(
            "Diameter",
            2. * radius,
            [center, vertex],
            SketchPlacementBinding::Diameter(sides),
        ),
        field(
            "Angle",
            vector[1].atan2(vector[0]),
            [center, vertex],
            SketchPlacementBinding::SteeringAngle,
        ),
    ]
}

fn polygon_members(sides: usize, radius: f64) -> Vec<SketchPlacementMember> {
    let mut members = chain_members(sides, true);
    members.push(SketchPlacementMember {
        geometry: SketchPlacementGeometry::Circle {
            center: sides,
            radius,
        },
        construction: true,
    });
    members
}
