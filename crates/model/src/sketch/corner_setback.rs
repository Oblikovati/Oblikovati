// SPDX-License-Identifier: GPL-2.0-only
use super::{line_residual::unit_direction, residual::point};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

pub(super) fn measure(variables: &[DualNumber]) -> Result<DualNumber, ResidualError> {
    let corner = point(variables, 0)?;
    let trim = point(variables, 2)?.minus(&corner)?;
    let ray = point(variables, 4)?.minus(&corner)?;
    let length = ray.length()?;
    let projection = unit_direction(&ray)?.dot(&trim)?;
    if projection.value() <= 0.0 || projection.value() >= length.value() {
        return Err(ResidualError::Domain(format!(
            "corner setback {} on segment {}: expected the trim point strictly between the corner and far endpoint",
            projection.value(),
            length.value()
        )));
    }
    Ok(projection)
}

pub(super) fn bounded_line(
    variables: &[DualNumber],
    trimmed_first: bool,
) -> Result<Vec<DualNumber>, ResidualError> {
    let order = if trimmed_first {
        [0, 1, 2, 3, 4, 5]
    } else {
        [0, 1, 4, 5, 2, 3]
    };
    let oriented: Vec<_> = order
        .into_iter()
        .map(|index| variables[index].clone())
        .collect();
    measure(&oriented)?;
    super::line_residual::point_on_line(variables)
}

pub(super) fn tangent(
    variables: &[DualNumber],
    side: super::LineContactSide,
    trimmed_first: bool,
) -> Result<Vec<DualNumber>, ResidualError> {
    let corner = point(variables, 0)?;
    let ray = unit_direction(&point(variables, 2)?.minus(&corner)?)?;
    let trim = point(variables, 4)?;
    let center = point(variables, 6)?;
    let line_side = ray.cross(&center.minus(&corner)?)?.value();
    let orientation = if trimmed_first { 1.0 } else { -1.0 };
    let desired = if side == super::LineContactSide::Left {
        1.0
    } else {
        -1.0
    };
    if line_side * orientation * desired <= 0.0 {
        return Err(ResidualError::Domain(format!(
            "corner tangent side {line_side}: expected the recorded nonzero side {side:?}"
        )));
    }
    Ok(vec![ray.dot(&center.minus(&trim)?)?])
}
