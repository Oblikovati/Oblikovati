// SPDX-License-Identifier: GPL-2.0-only
use super::residual::point;
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

pub(super) fn point_on_circle(variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
    positive_radius(&variables[4])?;
    let distance = point(variables, 0)?
        .minus(&point(variables, 2)?)?
        .length()?;
    Ok(vec![distance.minus(&variables[4])?])
}
pub(super) fn equal_radius(variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
    positive_radius(&variables[0])?;
    positive_radius(&variables[1])?;
    Ok(vec![variables[0].minus(&variables[1])?])
}
pub(super) fn positive_radius(radius: &DualNumber) -> Result<(), ResidualError> {
    if radius.value() <= 0.0 {
        return Err(ResidualError::Domain(format!(
            "circle radius {:?}: expected a strictly positive working-unit length",
            radius.value()
        )));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../../tests/sketch/circle_residuals.rs"]
mod tests;
