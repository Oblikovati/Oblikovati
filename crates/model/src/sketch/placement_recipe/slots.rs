// SPDX-License-Identifier: GPL-2.0-only
use super::fields::{distance, field};
use super::*;
use crate::sketch::SketchSlotPlacement;
use SketchPlacementBinding::{Diameter, Distance, SteeringAngle};
impl SketchPlacementRecipe {
    /// Build the constrained straight-slot graph; example: `SketchPlacementRecipe::straight_slot([[0.0,0.0],[10.0,0.0]],2.0)?`.
    pub fn straight_slot(centers: [[f64; 2]; 2], width: f64) -> Result<Self, SketchError> {
        let layout = SketchSlotPlacement::Centers(centers).layout(width)?;
        let mut members: Vec<_> = layout.members.into_iter().map(boundary).collect();
        members.push(SketchPlacementMember {
            path: SketchPathMember::Line([5, 4]),
            construction: true,
        });
        let mut relations: Vec<_> = [(0, 1), (2, 1), (2, 3), (0, 3)]
            .into_iter()
            .map(|(line, curve)| SketchPlacementRelation::LineTangent { line, curve })
            .collect();
        relations.push(SketchPlacementRelation::EqualRadius([1, 3]));
        let fields = straight_fields(centers, width, &layout.points);
        Ok(Self {
            points: layout.points,
            members,
            relations,
            fields,
        })
    }
    /// Build a shared-center arc slot without a redundant centerline arc; example: `SketchPlacementRecipe::arc_slot(center,[start,end],2.0,true)?`.
    pub fn arc_slot(
        center: [f64; 2],
        endpoints: [[f64; 2]; 2],
        width: f64,
        ccw: bool,
    ) -> Result<Self, SketchError> {
        let layout = SketchSlotPlacement::Arc {
            center,
            endpoints,
            counter_clockwise: ccw,
        }
        .layout(width)?;
        let points: Vec<_> = [4, 0, 2, 3, 1, 7, 6]
            .map(|index| layout.points[index])
            .to_vec();
        // Go aa8297d1: one shared center removes the need for a concentric relation.
        let members = [
            arc(0, [1, 2], ccw),
            arc(6, [2, 3], ccw),
            arc(0, [3, 4], !ccw),
            arc(5, [4, 1], ccw),
        ]
        .map(boundary)
        .to_vec();
        // Four pinned tangencies are independent; cap equality and a centerline arc add redundant equations.
        let relations = [[0, 1], [0, 3], [2, 1], [2, 3]]
            .map(SketchPlacementRelation::CircularTangent)
            .to_vec();
        let fields = curved_fields(center, endpoints, width, points[1]);
        Ok(Self {
            points,
            members,
            relations,
            fields,
        })
    }
}
fn boundary(path: SketchPathMember) -> SketchPlacementMember {
    SketchPlacementMember {
        path,
        construction: false,
    }
}
fn arc(center: usize, endpoints: [usize; 2], counter_clockwise: bool) -> SketchPathMember {
    SketchPathMember::Arc {
        center,
        endpoints,
        counter_clockwise,
    }
}
fn straight_fields(
    centers: [[f64; 2]; 2],
    width: f64,
    points: &[[f64; 2]],
) -> Vec<SketchPlacementField> {
    let delta = [centers[1][0] - centers[0][0], centers[1][1] - centers[0][1]];
    vec![
        field("Length", distance(centers), centers, Distance([5, 4])),
        field("Angle", delta[1].atan2(delta[0]), centers, SteeringAngle),
        field("Width", width, [points[0], points[3]], Diameter(3)),
    ]
}
fn curved_fields(
    center: [f64; 2],
    [start, end]: [[f64; 2]; 2],
    width: f64,
    outer_start: [f64; 2],
) -> Vec<SketchPlacementField> {
    let from = (start[1] - center[1]).atan2(start[0] - center[0]);
    let to = (end[1] - center[1]).atan2(end[0] - center[0]);
    vec![
        field(
            "Radius",
            distance([center, start]),
            [center, start],
            Distance([0, 5]),
        ),
        field("Sweep", to - from, [start, end], SteeringAngle),
        field("Width", width, [start, outer_start], Diameter(3)),
    ]
}
