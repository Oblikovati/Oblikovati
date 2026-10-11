// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{AngleTurn, LineContactSide, TangentDistanceContact, TangentLean};
impl SketchDimensionalConstraintCreation {
    pub(crate) fn validate(&self, geometry: &Sketch2D) -> Result<(), SketchError> {
        let relation = match self {
            Self::Measured(input) => input.relation(0),
            Self::Seeded(input) => input.ownership_relation(geometry)?,
        };
        // #3787: ownership-only records consume no IDs and never select or publish a geometric branch.
        geometry.validate_constraint_relation(&relation)
    }
}
impl SeededDimensionOperands {
    fn ownership_relation(&self, geometry: &Sketch2D) -> Result<PlanarConstraint, SketchError> {
        Ok(match *self {
            Self::OffsetSpline { offset } => offset_spline_ownership_relation(offset),
            Self::Angle(lines) => angle_ownership_relation(lines),
            Self::ThreePointAngle([vertex, first, second]) => {
                three_point_ownership_relation(vertex, [first, second])
            }
            Self::Offset {
                point_line: [point, line],
                linear_diameter,
            } => offset_ownership_relation(point, line, linear_diameter),
            Self::TangentDistance {
                line_curve: [line, curve],
                options,
            } => tangent_ownership_relation(geometry, line, curve, options)?,
        })
    }
}
fn tangent_ownership_relation(
    geometry: &Sketch2D,
    line: SketchEntityId,
    curve: SketchEntityId,
    options: TangentDistanceOptions,
) -> Result<PlanarConstraint, SketchError> {
    let lean = geometry
        .shared_line_contact(line, curve)?
        .map(|_| TangentLean::Positive);
    Ok(PlanarConstraint::TangentDistance {
        line,
        curve,
        parameter: 0,
        options,
        contact: TangentDistanceContact {
            side: LineContactSide::Left,
            lean,
        },
    })
}

fn angle_ownership_relation(lines: [SketchEntityId; 2]) -> PlanarConstraint {
    PlanarConstraint::Angle {
        lines,
        parameter: 0,
        turn: AngleTurn::CounterClockwise,
    }
}
fn three_point_ownership_relation(
    vertex: SketchEntityId,
    points: [SketchEntityId; 2],
) -> PlanarConstraint {
    PlanarConstraint::ThreePointAngle {
        vertex,
        points,
        parameter: 0,
        turn: AngleTurn::CounterClockwise,
    }
}

fn offset_ownership_relation(
    point: SketchEntityId,
    line: SketchEntityId,
    linear_diameter: bool,
) -> PlanarConstraint {
    PlanarConstraint::Offset {
        point,
        line,
        parameter: 0,
        side: LineContactSide::Left,
        linear_diameter,
    }
}

fn offset_spline_ownership_relation(offset: SketchEntityId) -> PlanarConstraint {
    PlanarConstraint::OffsetSplineDistance {
        offset,
        parameter: 0,
        side: LineContactSide::Left,
    }
}
