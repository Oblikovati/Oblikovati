// SPDX-License-Identifier: GPL-2.0-only
use super::{arc_sweep::radial_sweep, residual::point};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};
pub(super) fn length_residual(
    variables: &[DualNumber],
    target: f64,
    counter_clockwise: bool,
) -> Result<Vec<DualNumber>, ResidualError> {
    Ok(vec![
        length_measure(variables, counter_clockwise)?.shifted(-target)?,
    ])
}
pub(super) fn length_measure(
    variables: &[DualNumber],
    counter_clockwise: bool,
) -> Result<DualNumber, ResidualError> {
    let center = point(variables, 0)?;
    let start = point(variables, 2)?.minus(&center)?;
    let end = point(variables, 4)?.minus(&center)?;
    let sweep = radial_sweep(&start, &end, counter_clockwise)?;
    let magnitude = if counter_clockwise {
        sweep
    } else {
        sweep.negated()?
    };
    // Go AddArcLength takes the minor unsigned angle; authored major sweeps must retain their full length (#3713).
    Ok(start.length()?.product(&magnitude)?)
}
#[cfg(test)]
#[path = "../../tests/sketch/arc_length_residuals.rs"]
mod tests;
