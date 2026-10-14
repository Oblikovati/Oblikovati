// SPDX-License-Identifier: GPL-2.0-only
use super::fields::{distance, field};
use super::*;
use crate::sketch::SketchRectangleSeed;
use SketchPlacementBinding::{Distance, HorizontalDistance, SteeringAngle, VerticalDistance};
impl SketchPlacementRecipe {
    /// Build the two-corner constrained rectangle; example: `SketchPlacementRecipe::rectangle([[0.,0.],[10.,8.]])?`.
    pub fn rectangle([a, c]: [[f64; 2]; 2]) -> Result<Self, SketchError> {
        axis_rectangle(SketchRectangleSeed::Corners([a, c]).vertices()?.to_vec())
    }
    /// Keep the centre on a construction diagonal; example: `SketchPlacementRecipe::center_rectangle([0.,0.],[5.,4.])?`.
    pub fn center_rectangle(center: [f64; 2], corner: [f64; 2]) -> Result<Self, SketchError> {
        let points = SketchRectangleSeed::DirectedCenter([center, corner]).vertices()?;
        let mut recipe = axis_rectangle(points.to_vec())?;
        recipe.points.push(center);
        recipe
            .members
            .extend([[0, 2], [1, 3]].map(|points| SketchPlacementMember {
                geometry: SketchPlacementGeometry::Path(SketchPathMember::Line(points)),
                construction: true,
            }));
        recipe
            .relations
            .push(SketchPlacementRelation::Midpoint { point: 4, line: 4 });
        Ok(recipe)
    }
    /// Offset a base by the third point's perpendicular depth; example: `SketchPlacementRecipe::three_point_rectangle([[0.,0.],[10.,0.]],[3.,8.])?`.
    pub fn three_point_rectangle(
        base: [[f64; 2]; 2],
        height: [f64; 2],
    ) -> Result<Self, SketchError> {
        let [a, b, c, d] =
            SketchRectangleSeed::ThreePoints([base[0], base[1], height]).vertices()?;
        let delta = [b[0] - a[0], b[1] - a[1]];
        checked_rectangle(Self {
            circular_checks: Vec::new(),
            points: vec![a, b, c, d],
            members: super::geometry::chain_members(4, true),
            // Go aa8297d1 (#2014): the fourth perpendicular relation is redundant in a closed loop.
            relations: [[0, 1], [1, 2], [2, 3]]
                .map(SketchPlacementRelation::Perpendicular)
                .to_vec(),
            fields: vec![
                field("Length", distance(base), base, Distance([0, 1])),
                field("Angle", delta[1].atan2(delta[0]), base, SteeringAngle),
                field("Width", distance([b, c]), [b, c], Distance([1, 2])),
            ],
        })
    }
}
fn axis_rectangle(points: Vec<[f64; 2]>) -> Result<SketchPlacementRecipe, SketchError> {
    let [a, b, c] = [points[0], points[1], points[2]];
    use SketchPlacementRelation::{LineHorizontal, LineVertical};
    checked_rectangle(SketchPlacementRecipe {
        circular_checks: Vec::new(),
        points,
        members: super::geometry::chain_members(4, true),
        relations: vec![
            LineHorizontal(0),
            LineHorizontal(2),
            LineVertical(1),
            LineVertical(3),
        ],
        fields: vec![
            field(
                "Width",
                (c[0] - a[0]).abs(),
                [a, b],
                HorizontalDistance([0, 1]),
            ),
            field(
                "Height",
                (c[1] - a[1]).abs(),
                [b, c],
                VerticalDistance([1, 2]),
            ),
        ],
    })
}
fn checked_rectangle(recipe: SketchPlacementRecipe) -> Result<SketchPlacementRecipe, SketchError> {
    recipe.validate()?;
    for i in 0..4 {
        let edge = [recipe.points[i], recipe.points[(i + 1) % 4]];
        let length = distance(edge);
        if !length.is_finite() || length == 0. {
            return Err(SketchError::Definition(format!(
                "rectangle edge {edge:?}: expected a finite nonzero length"
            )));
        }
    }
    Ok(recipe)
}
