// SPDX-License-Identifier: GPL-2.0-only
use super::{ColumnPivotedQr, LinearAlgebraError, ResidualEvaluation, normalized_jacobian};
/// Classify local coordinate freedom using the normalized Jacobian row space.
/// Rank and squared projection thresholds are explicit; this does not prove finite motion.
/// Example: `let movable = coordinate_freedoms(&evaluation, 1e-9, 1e-6)?;`.
pub fn coordinate_freedoms(
    evaluation: &ResidualEvaluation,
    rank_tolerance: f64,
    squared_freedom_tolerance: f64,
) -> Result<Vec<bool>, LinearAlgebraError> {
    validate_freedom_tolerance(squared_freedom_tolerance)?;
    let columns = normalized_jacobian(evaluation)?.transpose();
    let qr = ColumnPivotedQr::factor(&columns, rank_tolerance)?;
    Ok(qr
        .coordinate_projection_residuals()?
        .into_iter()
        .map(|residual| residual > squared_freedom_tolerance)
        .collect())
}
fn validate_freedom_tolerance(tolerance: f64) -> Result<(), LinearAlgebraError> {
    if !tolerance.is_finite() || tolerance <= 0.0 || tolerance >= 1.0 {
        return Err(LinearAlgebraError(format!(
            "squared coordinate-freedom tolerance {tolerance:?}: expected a finite value strictly between 0 and 1"
        )));
    }
    Ok(())
}
