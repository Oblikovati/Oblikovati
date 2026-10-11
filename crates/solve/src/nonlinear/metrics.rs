// SPDX-License-Identifier: GPL-2.0-only

use super::{SolveError, checked};
use crate::residual::ResidualEvaluation;

use crate::row_scaling::RowScale;
pub(super) struct ScaledEvaluation {
    pub residuals: Vec<f64>,
    pub jacobian: Vec<Vec<f64>>,
    pub weights: Vec<RowScale>,
    pub model_scale: f64,
}

pub(super) fn model_scale(values: &[f64]) -> f64 {
    let scale = values
        .iter()
        .fold(0.0_f64, |scale, value| scale.max(value.abs()));
    if scale == 0.0 { 1.0 } else { scale }
}

pub(super) fn scale_evaluation(
    evaluation: &ResidualEvaluation,
    model_scale: f64,
) -> Result<ScaledEvaluation, SolveError> {
    let weights: Vec<RowScale> = evaluation
        .jacobian()
        .iter()
        .map(|row| RowScale::new(row))
        .collect();
    let residuals = scaled_residuals(evaluation, &weights, model_scale)?;
    let jacobian = evaluation
        .jacobian()
        .iter()
        .zip(&weights)
        .map(|(row, weight)| weight.row(row))
        .collect();
    Ok(ScaledEvaluation {
        residuals,
        jacobian,
        weights,
        model_scale,
    })
}

pub(super) fn scaled_residuals(
    evaluation: &ResidualEvaluation,
    weights: &[RowScale],
    model_scale: f64,
) -> Result<Vec<f64>, SolveError> {
    evaluation
        .values()
        .iter()
        .zip(weights)
        .map(|(&value, weight)| {
            checked(
                weight.displacement(value, model_scale),
                "normalized residual",
            )
        })
        .collect()
}

pub(super) fn norm(values: &[f64]) -> Result<f64, SolveError> {
    checked(
        values
            .iter()
            .fold(0.0_f64, |norm, value| norm.hypot(*value)),
        "residual norm",
    )
}

pub(super) fn infinity_norm(values: &[f64]) -> f64 {
    values
        .iter()
        .fold(0.0_f64, |largest, value| largest.max(value.abs()))
}

pub(super) fn conflicts(
    evaluation: &ResidualEvaluation,
    residuals: &[f64],
    tolerance: f64,
) -> Vec<usize> {
    let mut sources: Vec<(usize, f64)> = evaluation
        .source_rows()
        .iter()
        .enumerate()
        .map(|(source, rows)| (source, infinity_norm(&residuals[rows.clone()])))
        .filter(|(_, severity)| *severity > tolerance)
        .collect();
    sources.sort_by(|left, right| right.1.total_cmp(&left.1).then(left.0.cmp(&right.0)));
    sources.into_iter().map(|(source, _)| source).collect()
}

pub(super) fn convergence_residuals(
    evaluation: &ResidualEvaluation,
    blocks: &[crate::residual::ResidualBlock<'_>],
    scaled: &ScaledEvaluation,
) -> Vec<f64> {
    let mut residuals = scaled.residuals.clone();
    for (block, rows) in blocks.iter().zip(evaluation.source_rows()) {
        if block.source.tolerance_kind() == crate::residual::ResidualTolerance::Unitless {
            residuals[rows.clone()].copy_from_slice(&evaluation.values()[rows.clone()]);
        }
    }
    residuals
}
