// SPDX-License-Identifier: GPL-2.0-only

use super::{SolveError, checked};
use crate::residual::{ResidualBlock, ResidualEvaluation, ResidualTolerance};

use crate::row_scaling::RowScale;
pub(super) struct ScaledEvaluation {
    pub residuals: Vec<f64>,
    pub jacobian: Vec<Vec<f64>>,
    pub weights: Vec<RowScale>,
    pub model_scale: f64,
    pub domains: Vec<ResidualTolerance>,
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
    blocks: &[ResidualBlock<'_>],
) -> Result<ScaledEvaluation, SolveError> {
    let weights: Vec<RowScale> = evaluation
        .jacobian()
        .iter()
        .map(|row| RowScale::new(row))
        .collect();
    let domains = equation_domains(evaluation, blocks);
    let residuals = scaled_residuals(evaluation, &weights, model_scale, &domains)?;
    let jacobian = working_jacobian(evaluation, &weights, model_scale, &domains)?;
    Ok(ScaledEvaluation {
        residuals,
        jacobian,
        weights,
        model_scale,
        domains,
    })
}

pub(super) fn scaled_residuals(
    evaluation: &ResidualEvaluation,
    weights: &[RowScale],
    model_scale: f64,
    domains: &[ResidualTolerance],
) -> Result<Vec<f64>, SolveError> {
    evaluation
        .values()
        .iter()
        .zip(weights)
        .zip(domains)
        .map(|((&value, weight), domain)| {
            checked(
                match domain {
                    ResidualTolerance::ModelRelative => weight.displacement(value, model_scale),
                    ResidualTolerance::ModelLength => value / model_scale,
                    ResidualTolerance::Unitless => value,
                },
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

fn equation_domains(
    evaluation: &ResidualEvaluation,
    blocks: &[ResidualBlock<'_>],
) -> Vec<ResidualTolerance> {
    blocks
        .iter()
        .zip(evaluation.source_rows())
        .flat_map(|(block, rows)| (0..rows.len()).map(|row| block.source.equation_tolerance(row)))
        .collect()
}
fn working_jacobian(
    evaluation: &ResidualEvaluation,
    weights: &[RowScale],
    model_scale: f64,
    domains: &[ResidualTolerance],
) -> Result<Vec<Vec<f64>>, SolveError> {
    evaluation
        .jacobian()
        .iter()
        .zip(weights)
        .zip(domains)
        .map(|((row, weight), domain)| match domain {
            ResidualTolerance::ModelRelative => Ok(weight.row(row)),
            ResidualTolerance::ModelLength => Ok(row.clone()),
            ResidualTolerance::Unitless => row
                .iter()
                .map(|value| checked(value * model_scale, "unitless working derivative"))
                .collect(),
        })
        .collect()
}
