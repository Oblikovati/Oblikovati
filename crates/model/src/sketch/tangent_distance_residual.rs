// SPDX-License-Identifier: GPL-2.0-only
use super::{AngleTurn, LineContactSide, TangentDistanceContact, TangentLean};
use super::{
    circle_residual::positive_radius,
    tangent_residual::{radial_projection, signed_line_distance},
};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};
pub(super) fn oriented_distance(
    variables: &[DualNumber],
    side: LineContactSide,
) -> Result<DualNumber, ResidualError> {
    positive_radius(&variables[6])?;
    let distance = signed_line_distance(variables)?.scaled(if side == LineContactSide::Left {
        1.0
    } else {
        -1.0
    })?;
    if distance.value() < 0.0 {
        return Err(ResidualError::Domain(format!(
            "tangent-distance side {side:?} distance {:?}: expected the recorded nonnegative center side",
            distance.value()
        )));
    }
    Ok(distance)
}
pub(super) fn distance_residual(
    variables: &[DualNumber],
    target: f64,
    far_side: bool,
    contact: TangentDistanceContact,
    at_b: Option<bool>,
) -> Result<Vec<DualNumber>, ResidualError> {
    let distance = oriented_distance(variables, contact.side)?;
    require_lean(variables, contact.lean, at_b)?;
    if !far_side
        && target == 0.0
        && let Some(at_b) = at_b
    {
        // The metric row has a double root at a shared tangent point (Go #2014).
        return super::tangent_residual::pinned_line_tangent(variables, at_b, contact.side);
    }
    let offset = circular_offset(&distance, &variables[6], far_side)?;
    Ok(vec![offset.shifted(-target)?])
}
fn require_lean(
    variables: &[DualNumber],
    lean: Option<TangentLean>,
    at_b: Option<bool>,
) -> Result<(), ResidualError> {
    let (Some(lean), Some(at_b)) = (lean, at_b) else {
        return Ok(());
    };
    let dot = radial_projection(variables, at_b)?.value();
    let sign = if lean == TangentLean::Positive {
        1.0
    } else {
        -1.0
    };
    if dot * sign < 0.0 {
        return Err(ResidualError::Domain(format!(
            "tangent-distance lean {lean:?} projection {dot:?}: expected the recorded radial half-plane"
        )));
    }
    Ok(())
}
pub(super) fn angular_residual(
    variables: &[DualNumber],
    target: f64,
    at_b: bool,
    side: LineContactSide,
) -> Result<Vec<DualNumber>, ResidualError> {
    oriented_distance(variables, side)?;
    let touch = if at_b { 2 } else { 0 };
    let mut rays = variables[..4].to_vec();
    rays.extend_from_slice(&variables[touch..touch + 2]);
    rays.extend_from_slice(&variables[4..6]);
    let turn = if side == LineContactSide::Left {
        AngleTurn::CounterClockwise
    } else {
        AngleTurn::Clockwise
    };
    super::angle::angle_residual(&rays, target, turn)
}

#[cfg(test)]
#[path = "../../tests/sketch/tangent_distance_residuals.rs"]
mod tests;

pub(super) fn circular_offset(
    distance: &DualNumber,
    radius: &DualNumber,
    far_side: bool,
) -> Result<DualNumber, ResidualError> {
    if far_side {
        return Ok(distance.plus(radius)?);
    }
    Ok(distance.minus(radius)?)
}
