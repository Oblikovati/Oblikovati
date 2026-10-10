// SPDX-License-Identifier: GPL-2.0-only
use super::{
    CircleContact, LineContactSide, circle_residual::positive_radius,
    line_residual::unit_direction, residual::point,
};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

pub(super) fn line_tangent(
    variables: &[DualNumber],
    side: LineContactSide,
) -> Result<Vec<DualNumber>, ResidualError> {
    positive_radius(&variables[6])?;
    let signed_distance = signed_line_distance(variables)?;
    let sign = match side {
        LineContactSide::Left => 1.0,
        LineContactSide::Right => -1.0,
    };
    Ok(vec![signed_distance.scaled(sign)?.minus(&variables[6])?])
}
pub(super) fn signed_line_distance(variables: &[DualNumber]) -> Result<DualNumber, ResidualError> {
    let anchor = point(variables, 0)?;
    let direction = unit_direction(&point(variables, 2)?.minus(&anchor)?)?;
    Ok(direction.cross(&point(variables, 4)?.minus(&anchor)?)?)
}

pub(super) fn circle_tangent(
    variables: &[DualNumber],
    contact: CircleContact,
) -> Result<Vec<DualNumber>, ResidualError> {
    positive_radius(&variables[4])?;
    positive_radius(&variables[5])?;
    let distance = point(variables, 0)?
        .minus(&point(variables, 2)?)?
        .length()?;
    if distance.value() == 0.0 {
        return Err(ResidualError::Domain("circle center distance 0.0: expected distinct centers with a defined contact direction".into()));
    }
    let target = contact_distance(&variables[4], &variables[5], contact)?;
    Ok(vec![distance.minus(&target)?])
}
pub(super) fn contact_distance(
    first: &DualNumber,
    second: &DualNumber,
    contact: CircleContact,
) -> Result<DualNumber, ResidualError> {
    let target = match contact {
        CircleContact::External => first.plus(second)?,
        CircleContact::FirstContainsSecond => first.minus(second)?,
        CircleContact::SecondContainsFirst => second.minus(first)?,
    };
    if target.value() <= 0.0 {
        return Err(ResidualError::Domain(format!(
            "circle contact {contact:?} target {:?}, radii [{:?}, {:?}]: expected a positive gap with the recorded enclosing circle",
            target.value(),
            first.value(),
            second.value()
        )));
    }
    Ok(target)
}

#[cfg(test)]
#[path = "../../tests/sketch/tangent_residuals.rs"]
mod tests;

pub(super) fn pinned_line_tangent(
    variables: &[DualNumber],
    at_b: bool,
    side: LineContactSide,
) -> Result<Vec<DualNumber>, ResidualError> {
    require_shared_line_side(variables, side)?;
    // A distance-minus-derived-radius equation has a double root here (Go #2014).
    Ok(vec![radial_projection(variables, at_b)?])
}
pub(super) fn radial_projection(
    variables: &[DualNumber],
    at_b: bool,
) -> Result<DualNumber, ResidualError> {
    let anchor = point(variables, 0)?;
    let direction = unit_direction(&point(variables, 2)?.minus(&anchor)?)?;
    let touch = point(variables, if at_b { 2 } else { 0 })?;
    Ok(direction.dot(&point(variables, 4)?.minus(&touch)?)?)
}
pub(super) fn radial_contact_dot(variables: &[DualNumber]) -> Result<DualNumber, ResidualError> {
    let touch = point(variables, 6)?;
    let first = unit_direction(&touch.minus(&point(variables, 0)?)?)?;
    let second = unit_direction(&touch.minus(&point(variables, 2)?)?)?;
    Ok(first.dot(&second)?)
}
pub(super) fn pinned_circular_tangent(
    variables: &[DualNumber],
    contact: CircleContact,
) -> Result<Vec<DualNumber>, ResidualError> {
    require_shared_circular_branch(variables, contact)?;
    let touch = point(variables, 6)?;
    let direction = unit_direction(&touch.minus(&point(variables, 0)?)?)?;
    // Shared-point collinearity has a simple root and contributes rank at exact contact (#2014).
    Ok(vec![direction.cross(&point(variables, 2)?.minus(&touch)?)?])
}

fn require_shared_line_side(
    variables: &[DualNumber],
    side: LineContactSide,
) -> Result<(), ResidualError> {
    positive_radius(&variables[6])?;
    let sign = if side == LineContactSide::Left {
        1.0
    } else {
        -1.0
    };
    let signed = signed_line_distance(variables)?.value();
    if signed * sign <= 0.0 {
        return Err(ResidualError::Domain(format!(
            "shared line contact {side:?} signed distance {signed:?}: expected the recorded nonzero center side"
        )));
    }
    Ok(())
}

fn require_shared_circular_branch(
    variables: &[DualNumber],
    contact: CircleContact,
) -> Result<(), ResidualError> {
    positive_radius(&variables[4])?;
    positive_radius(&variables[5])?;
    contact_distance(&variables[4], &variables[5], contact)?;
    let dot = radial_contact_dot(variables)?.value();
    if (contact == CircleContact::External && dot > 0.0)
        || (contact != CircleContact::External && dot < 0.0)
    {
        return Err(ResidualError::Domain(format!(
            "shared circular contact {contact:?} radial dot {dot:?}: expected the recorded external/opposed or internal/aligned half-plane"
        )));
    }
    Ok(())
}
