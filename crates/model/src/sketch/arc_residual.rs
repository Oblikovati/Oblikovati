// SPDX-License-Identifier: GPL-2.0-only
use super::{circle_residual::positive_radius, residual::point};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

pub(super) fn circularity(variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
    let center = point(variables, 0)?;
    let start_radius = point(variables, 2)?.minus(&center)?.length()?;
    let end_radius = point(variables, 4)?.minus(&center)?.length()?;
    // Structural circularity must move End with radius edits (Go #1419), not repair it after solving.
    positive_radius(&start_radius)?;
    positive_radius(&end_radius)?;
    Ok(vec![end_radius.minus(&start_radius)?])
}

#[cfg(test)]
#[path = "../../tests/sketch/arc_residuals.rs"]
mod tests;
