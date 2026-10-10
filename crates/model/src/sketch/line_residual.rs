// SPDX-License-Identifier: GPL-2.0-only
use super::{LineRelation, residual::point};
use oblikovati_solve::{
    ad::{DualNumber, DualVector},
    residual::ResidualError,
};

pub(super) fn point_on_line(variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
    let anchor = point(variables, 2)?;
    let direction = unit_direction(&point(variables, 4)?.minus(&anchor)?)?;
    let offset = point(variables, 0)?.minus(&anchor)?;
    Ok(vec![direction.cross(&offset)?])
}
pub(super) fn line_relation(
    variables: &[DualNumber],
    relation: LineRelation,
) -> Result<Vec<DualNumber>, ResidualError> {
    let first = point(variables, 2)?.minus(&point(variables, 0)?)?;
    let second = point(variables, 6)?.minus(&point(variables, 4)?)?;
    match relation {
        LineRelation::EqualLength => Ok(vec![first.length()?.minus(&second.length()?)?]),
        LineRelation::Parallel => {
            let (first, second) = oriented_directions(&first, &second)?;
            Ok(vec![first.cross(&second)?])
        }
        LineRelation::Perpendicular => {
            let (first, second) = oriented_directions(&first, &second)?;
            Ok(vec![first.dot(&second)?])
        }
        LineRelation::Collinear => collinear(variables, &first, &second),
    }
}
fn oriented_directions(
    first: &DualVector<2>,
    second: &DualVector<2>,
) -> Result<(DualVector<2>, DualVector<2>), ResidualError> {
    Ok((unit_direction(first)?, unit_direction(second)?))
}
fn collinear(
    variables: &[DualNumber],
    first: &DualVector<2>,
    second: &DualVector<2>,
) -> Result<Vec<DualNumber>, ResidualError> {
    let (first, second) = oriented_directions(first, second)?;
    let offset = point(variables, 4)?.minus(&point(variables, 0)?)?;
    Ok(vec![first.cross(&second)?, first.cross(&offset)?])
}

fn unit_direction(direction: &DualVector<2>) -> Result<DualVector<2>, ResidualError> {
    let length = direction.length()?;
    if length.value() == 0.0 {
        let values: Vec<f64> = direction
            .components()
            .iter()
            .map(DualNumber::value)
            .collect();
        return Err(ResidualError::Domain(format!(
            "line direction {values:?}: expected a nonzero direction with a defined orientation"
        )));
    }
    // Normalize each component before products; |a|*|b| can overflow or underflow (#3705).
    Ok(DualVector::new([
        direction.components()[0].quotient(&length)?,
        direction.components()[1].quotient(&length)?,
    ])?)
}

#[cfg(test)]
#[path = "../../tests/sketch/line_residuals.rs"]
mod tests;
