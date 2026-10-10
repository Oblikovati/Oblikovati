// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchError,
    line_residual::unit_direction, residual::point,
};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};
use std::f64::consts::{PI, TAU};
/// Authored angle direction from the first ray to the second; retained after edits.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case", deny_unknown_fields)]
pub enum AngleTurn {
    CounterClockwise,
    Clockwise,
}
impl Sketch2D {
    /// Retain the seed turn between oriented lines (CCW wins collinear ties); example: `sketch.add_angle(id, [a, b], parameter)?`.
    pub fn add_angle(
        &mut self,
        id: SketchConstraintId,
        lines: [SketchEntityId; 2],
        parameter: u64,
    ) -> Result<(), SketchError> {
        let first = self.line(lines[0])?.endpoints;
        let second = self.line(lines[1])?.endpoints;
        let turn = self.angle_seed_turn(id, [first[0], first[1], second[0], second[1]])?;
        self.add_constraint(
            id,
            PlanarConstraint::Angle {
                lines,
                parameter,
                turn,
            },
        )
    }
    /// Retain the turn around a shared vertex; example: `sketch.add_three_point_angle(id, vertex, [a, b], parameter)?`.
    pub fn add_three_point_angle(
        &mut self,
        id: SketchConstraintId,
        vertex: SketchEntityId,
        points: [SketchEntityId; 2],
        parameter: u64,
    ) -> Result<(), SketchError> {
        let turn = self.angle_seed_turn(id, [vertex, points[0], vertex, points[1]])?;
        self.add_constraint(
            id,
            PlanarConstraint::ThreePointAngle {
                vertex,
                points,
                parameter,
                turn,
            },
        )
    }
    fn angle_seed_turn(
        &self,
        id: SketchConstraintId,
        points: [SketchEntityId; 4],
    ) -> Result<AngleTurn, SketchError> {
        let variables =
            self.point_constants(&points, |cause| angle_selection_failure(id, cause))?;
        let angle =
            angle_measure(&variables).map_err(|cause| angle_selection_failure(id, cause))?;
        Ok(if angle.value() < 0.0 && angle.value() > -PI {
            AngleTurn::Clockwise
        } else {
            AngleTurn::CounterClockwise
        })
    }
}
pub(super) fn angle_residual(
    variables: &[DualNumber],
    target: f64,
    turn: AngleTurn,
) -> Result<Vec<DualNumber>, ResidualError> {
    let sign = if turn == AngleTurn::CounterClockwise {
        1.0
    } else {
        -1.0
    };
    let difference = angle_measure(variables)?.shifted(-sign * target)?;
    // A signed row keeps rank at parallel/straight angles where Go's abs(cross) is stationary.
    Ok(vec![wrap_angle_difference(difference)?])
}
fn angle_measure(variables: &[DualNumber]) -> Result<DualNumber, ResidualError> {
    let first = unit_direction(&point(variables, 2)?.minus(&point(variables, 0)?)?)?;
    let second = unit_direction(&point(variables, 6)?.minus(&point(variables, 4)?)?)?;
    Ok(first.cross(&second)?.atan2(&first.dot(&second)?)?)
}
fn wrap_angle_difference(difference: DualNumber) -> Result<DualNumber, ResidualError> {
    if difference.value() > PI {
        return Ok(difference.shifted(-TAU)?);
    }
    if difference.value() < -PI {
        return Ok(difference.shifted(TAU)?);
    }
    Ok(difference)
}
#[cfg(test)]
#[path = "../../tests/sketch/angle_residuals.rs"]
mod tests;

fn angle_selection_failure(constraint: SketchConstraintId, cause: ResidualError) -> SketchError {
    SketchError::AngleSelection { constraint, cause }
}
