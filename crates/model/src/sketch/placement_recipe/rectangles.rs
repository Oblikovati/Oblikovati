// SPDX-License-Identifier: GPL-2.0-only
use super::fields::{distance, field};
use super::*;
use crate::sketch::SketchRectangleSeed;
use SketchPlacementBinding::{Distance, HorizontalDistance, LineAngle, VerticalDistance};
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
                centerline: false,
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
            picked_points: Default::default(),
            circular_checks: Vec::new(),
            points: vec![a, b, c, d],
            members: super::geometry::chain_members(4, true),
            // Go aa8297d1 (#2014): the fourth perpendicular relation is redundant in a closed loop.
            relations: [[0, 1], [1, 2], [2, 3]]
                .map(SketchPlacementRelation::Perpendicular)
                .to_vec(),
            fields: vec![
                field("Length", distance(base), base, Distance([0, 1])),
                field("Angle", delta[1].atan2(delta[0]), base, LineAngle(0)),
                field("Width", distance([b, c]), [b, c], Distance([1, 2])),
            ],
        })
    }
    /// A rectangle from its center, the middle of one side and a third point setting its width,
    /// with construction diagonals crossing at the center.
    /// Example: `SketchPlacementRecipe::three_point_center_rectangle([center, side], third)`.
    pub fn three_point_center_rectangle(
        [center, side]: [[f64; 2]; 2],
        third: [f64; 2],
    ) -> Result<Self, SketchError> {
        let half = [side[0] - center[0], side[1] - center[1]];
        let start = [center[0] - half[0], center[1] - half[1]];
        let [_, _, far, _] = SketchRectangleSeed::ThreePoints([start, side, third]).vertices()?;
        let depth = [far[0] - side[0], far[1] - side[1]];
        let corner = |along: f64, across: f64| {
            [
                center[0] + along * half[0] + across * depth[0],
                center[1] + along * half[1] + across * depth[1],
            ]
        };
        let (a, b, c, d) = (
            corner(-1., -1.),
            corner(1., -1.),
            corner(1., 1.),
            corner(-1., 1.),
        );
        let mut members = super::geometry::chain_members(4, true);
        members.extend([[0, 2], [1, 3]].map(|points| SketchPlacementMember {
            centerline: false,
            geometry: SketchPlacementGeometry::Path(SketchPathMember::Line(points)),
            construction: true,
        }));
        checked_rectangle(Self {
            picked_points: Default::default(),
            circular_checks: Vec::new(),
            points: vec![a, b, c, d],
            members,
            relations: [[0, 1], [1, 2], [2, 3]]
                .map(SketchPlacementRelation::Perpendicular)
                .to_vec(),
            fields: vec![
                field("Length", distance([a, b]), [a, b], Distance([0, 1])),
                field("Angle", half[1].atan2(half[0]), [a, b], LineAngle(0)),
                field("Width", distance([b, c]), [b, c], Distance([1, 2])),
            ],
        })
    }
}
fn axis_rectangle(points: Vec<[f64; 2]>) -> Result<SketchPlacementRecipe, SketchError> {
    let [a, b, c] = [points[0], points[1], points[2]];
    use SketchPlacementRelation::{LineHorizontal, LineVertical};
    checked_rectangle(SketchPlacementRecipe {
        picked_points: Default::default(),
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
