// SPDX-License-Identifier: GPL-2.0-only

use super::{
    SolveError, checked,
    metrics::{ScaledEvaluation, norm},
};
use crate::linalg::{ColumnPivotedQr, DenseMatrix};

pub(super) const INITIAL_DAMPING: f64 = 1e-3; // tol:numeric — Go LM warm start (#3703).
pub(super) const MAX_DAMPING: f64 = 1e15; // tol:numeric — finite trust budget from Go.
const MIN_DAMPING: f64 = 1e-12; // tol:numeric — prevents loss of regularization.
const STEP_RANK_TOLERANCE: f64 = 1e-14; // tol:numeric — working precision of regularized QR.

pub(super) fn damped_delta(
    scaled: &ScaledEvaluation,
    columns: usize,
    damping: f64,
) -> Result<Vec<f64>, SolveError> {
    let observed: Vec<usize> = (0..columns)
        .filter(|&column| scaled.jacobian.iter().any(|row| row[column] != 0.0))
        .collect();
    let mut delta = vec![0.0; columns];
    if observed.is_empty() {
        return Ok(delta);
    }
    for (&column, value) in observed
        .iter()
        .zip(observed_step(scaled, &observed, damping)?)
    {
        delta[column] = value;
    }
    Ok(delta)
}

fn observed_step(
    scaled: &ScaledEvaluation,
    columns: &[usize],
    damping: f64,
) -> Result<Vec<f64>, SolveError> {
    // A zero working-Jacobian column has only its damping equation, whose unique step is zero.
    // Eliminate it before QR so roundoff cannot move a free coordinate or radius (#3707).
    let mut rows: Vec<Vec<f64>> = scaled
        .jacobian
        .iter()
        .map(|row| columns.iter().map(|&column| row[column]).collect())
        .collect();
    let mut rhs: Vec<f64> = scaled.residuals.iter().map(|value| -value).collect();
    append_damping(&mut rows, columns.len(), damping)?;
    rhs.resize(rows.len(), 0.0);
    let matrix = DenseMatrix::new(columns.len(), rows)?;
    Ok(ColumnPivotedQr::factor(&matrix, STEP_RANK_TOLERANCE)?.least_squares(&rhs)?)
}

fn append_damping(
    rows: &mut Vec<Vec<f64>>,
    columns: usize,
    damping: f64,
) -> Result<(), SolveError> {
    let original_rows = rows.len();
    for column in 0..columns {
        let column_norm = rows[..original_rows]
            .iter()
            .fold(0.0_f64, |norm, row| norm.hypot(row[column]))
            .max(1.0);
        let mut row = vec![0.0; columns];
        row[column] = checked(damping.sqrt() * column_norm, "damping diagonal")?;
        rows.push(row);
    }
    Ok(())
}

pub(super) fn trial_values(
    values: &[f64],
    delta: &[f64],
    scale: f64,
) -> Result<Vec<f64>, SolveError> {
    values
        .iter()
        .zip(delta)
        .map(|(&value, &change)| {
            if change == 0.0 {
                return Ok(value);
            }
            checked(
                value + checked(change * scale, "variable step")?,
                "trial variable",
            )
        })
        .collect()
}

pub(super) fn accepted_damping(
    scaled: &ScaledEvaluation,
    delta: &[f64],
    next: f64,
    damping: f64,
) -> Result<f64, SolveError> {
    let current = norm(&scaled.residuals)?;
    let ratio = predicted_norm(scaled, delta)? / current;
    let decrease = (1.0 - ratio) * (1.0 + ratio);
    let actual_ratio = next / current;
    let actual = (1.0 - actual_ratio) * (1.0 + actual_ratio);
    let factor = gain_factor(actual, decrease);
    Ok(checked(damping * factor, "accepted damping")?.max(MIN_DAMPING))
}

fn predicted_norm(scaled: &ScaledEvaluation, delta: &[f64]) -> Result<f64, SolveError> {
    let predicted: Vec<f64> = scaled
        .jacobian
        .iter()
        .zip(&scaled.residuals)
        .map(|(row, residual)| {
            row.iter()
                .zip(delta)
                .fold(*residual, |sum, (entry, change)| sum + entry * change)
        })
        .collect();
    norm(&predicted)
}

fn gain_factor(actual: f64, predicted: f64) -> f64 {
    if predicted <= 0.0 || actual >= predicted {
        return 1.0 / 3.0;
    }
    let gain = actual / predicted;
    (1.0 - (2.0 * gain - 1.0).powi(3)).max(1.0 / 3.0)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn gain_schedule_preserves_go_acceptance_policy_without_squared_norms() {
        assert_eq!(gain_factor(1.0, 0.0), 1.0 / 3.0);
        assert_eq!(gain_factor(1.0, -1.0), 1.0 / 3.0);
        assert_eq!(gain_factor(1.0, 1e-300), 1.0 / 3.0);
        assert_eq!(gain_factor(0.5, 1.0), 1.0);
        assert_eq!(gain_factor(0.25, 1.0), 1.125);
    }
    #[test]
    fn arithmetic_limits_refuse_unrepresentable_trial_coordinates() {
        assert!(trial_values(&[f64::MAX], &[1.0], f64::MAX).is_err());
        assert!(trial_values(&[0.0], &[f64::MAX], 2.0).is_err());
        assert_eq!(trial_values(&[3.0], &[-0.5], 2.0).unwrap(), [2.0]);
    }
    #[test]
    fn unused_column_has_an_exact_zero_regularized_step() {
        let scaled = ScaledEvaluation {
            residuals: vec![3.0],
            jacobian: vec![vec![1.0, 0.0]],
            weights: vec![],
            model_scale: 1.0,
            domains: vec![],
        };
        let delta = damped_delta(&scaled, 2, INITIAL_DAMPING).unwrap();
        assert_eq!(delta[1], 0.0, "unused column delta {:?}", delta[1]);
    }
}
