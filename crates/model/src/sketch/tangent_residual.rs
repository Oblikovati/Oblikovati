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
