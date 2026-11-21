// SPDX-License-Identifier: GPL-2.0-only

//! Local Jacobian structure; these results do not establish nonlinear convergence.
use crate::{
    linalg::{ColumnPivotedQr, DenseMatrix, LinearAlgebraError},
    residual::ResidualEvaluation,
};
use std::fmt;
mod freedom;
pub use freedom::coordinate_freedoms;

/// Go-compatible structural status; redundant rows take precedence over free variables.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ConstraintStatus {
    WellConstrained,
    UnderConstrained,
    OverConstrained,
}
impl fmt::Display for ConstraintStatus {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        output.write_str(match self {
            Self::WellConstrained => "well-constrained",
            Self::UnderConstrained => "under-constrained",
            Self::OverConstrained => "over-constrained",
        })
    }
}

/// Numerical freedom and row dependence at the captured evaluation point.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ConstraintStructure {
    pub variables: usize,
    pub equations: usize,
    pub rank: usize,
    pub degrees_of_freedom: usize,
    pub redundant_equations: usize,
    pub status: ConstraintStatus,
}

/// Analyze unit-normalized Jacobian rows at an explicit relative pivot tolerance.
/// Example: `let structure = analyze_structure(&evaluation, 1e-9)?;`.
pub fn analyze_structure(
    evaluation: &ResidualEvaluation,
    tolerance: f64,
) -> Result<ConstraintStructure, LinearAlgebraError> {
    let normalized = normalized_jacobian(evaluation)?;
    let rank = ColumnPivotedQr::factor(&normalized, tolerance)?.rank();
    Ok(structure(
        evaluation.variable_count(),
        evaluation.values().len(),
        rank,
    ))
}

/// Identify sources whose individual removal preserves the full numerical rank.
/// These are alternative removals, not a set that can safely be deleted together.
/// Example: `let candidates = redundant_sources(&evaluation, 1e-9)?;`.
pub fn redundant_sources(
    evaluation: &ResidualEvaluation,
    tolerance: f64,
) -> Result<Vec<usize>, LinearAlgebraError> {
    let normalized = normalized_jacobian(evaluation)?;
    let rank = ColumnPivotedQr::factor(&normalized, tolerance)?.rank();
    if normalized.rows().len() == rank {
        return Ok(vec![]);
    }
    let mut candidates = Vec::new();
    for (source, range) in evaluation.source_rows().iter().enumerate() {
        if range.is_empty() {
            continue;
        }
        let retained = normalized
            .rows()
            .iter()
            .enumerate()
            .filter(|(row, _)| !range.contains(row))
            .map(|(_, row)| row.clone())
            .collect();
        let reduced = DenseMatrix::new(normalized.columns(), retained)?;
        if ColumnPivotedQr::factor(&reduced, tolerance)?.rank() == rank {
            candidates.push(source);
        }
    }
    Ok(candidates)
}

fn normalized_jacobian(evaluation: &ResidualEvaluation) -> Result<DenseMatrix, LinearAlgebraError> {
    let rows = evaluation
        .jacobian()
        .iter()
        .map(|row| crate::row_scaling::RowScale::new(row).row(row))
        .collect();
    DenseMatrix::new(evaluation.variable_count(), rows)
}

fn structure(variables: usize, equations: usize, rank: usize) -> ConstraintStructure {
    let degrees_of_freedom = variables - rank;
    let redundant_equations = equations - rank;
    let status = if redundant_equations > 0 {
        ConstraintStatus::OverConstrained
    } else if degrees_of_freedom > 0 {
        ConstraintStatus::UnderConstrained
    } else {
        ConstraintStatus::WellConstrained
    };
    ConstraintStructure {
        variables,
        equations,
        rank,
        degrees_of_freedom,
        redundant_equations,
        status,
    }
}
