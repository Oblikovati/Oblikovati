// SPDX-License-Identifier: GPL-2.0-only
use super::line_residual::unit_direction;
use oblikovati_solve::{
    ad::{DualNumber, DualVector},
    residual::ResidualError,
};
use std::f64::consts::TAU;
pub(super) fn radial_sweep(
    start: &DualVector<2>,
    end: &DualVector<2>,
    counter_clockwise: bool,
) -> Result<DualNumber, ResidualError> {
    let first = unit_direction(start)?;
    let second = unit_direction(end)?;
    let angle = first.cross(&second)?.atan2(&first.dot(&second)?)?;
    directed_sweep(angle, counter_clockwise)
}
/// The unsigned turn from an arc's start to its end; variables are center, start and end.
pub(super) fn sweep_measure(
    variables: &[DualNumber],
    counter_clockwise: bool,
) -> Result<DualNumber, ResidualError> {
    let center = super::residual::point(variables, 0)?;
    let start = super::residual::point(variables, 2)?.minus(&center)?;
    let end = super::residual::point(variables, 4)?.minus(&center)?;
    let sweep = radial_sweep(&start, &end, counter_clockwise)?;
    if counter_clockwise {
        return Ok(sweep);
    }
    Ok(sweep.negated()?)
}
pub(super) fn sweep_residual(
    variables: &[DualNumber],
    target: f64,
    counter_clockwise: bool,
) -> Result<Vec<DualNumber>, ResidualError> {
    Ok(vec![
        sweep_measure(variables, counter_clockwise)?.shifted(-target)?,
    ])
}
fn directed_sweep(angle: DualNumber, counter_clockwise: bool) -> Result<DualNumber, ResidualError> {
    if angle.value() == 0.0 {
        return Err(ResidualError::Domain(format!(
            "arc directed angle {:?}: expected distinct endpoint directions; zero and full sweeps are ambiguous",
            angle.value()
        )));
    }
    if counter_clockwise && angle.value() < 0.0 {
        return Ok(angle.shifted(TAU)?);
    }
    if !counter_clockwise && angle.value() > 0.0 {
        return Ok(angle.shifted(-TAU)?);
    }
    Ok(angle)
}
