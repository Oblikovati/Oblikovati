// SPDX-License-Identifier: GPL-2.0-only

//! Bounded, immutable Levenberg–Marquardt iteration for the shared constraint core.
mod iteration;
mod metrics;
mod step;
use crate::residual::{ResidualBlock, ResidualEvaluation};
use crate::{
    diagnostics::ConstraintStructure, linalg::LinearAlgebraError, residual::EvaluationError,
};
use std::{error::Error, fmt};

/// Numerical policy, independent from the model's geometric tolerances.
#[derive(Clone, Copy, Debug)]
pub struct SolveOptions {
    pub max_iterations: usize,
    pub max_trial_steps: usize,
    pub relative_tolerance: f64,
    pub rank_tolerance: f64,
}
impl Default for SolveOptions {
    fn default() -> Self {
        Self {
            max_iterations: 100,
            max_trial_steps: 24,
            relative_tolerance: 1e-10, // tol:numeric — Go residual displacement target (#3703).
            rank_tolerance: 1e-9, // tol:numeric — relative working rank, separate from convergence.
        }
    }
}

/// Satisfaction and termination are independent from local constraint structure.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SolveTermination {
    Converged,
    IterationLimit,
    Stalled,
    Cancelled,
}

/// Final accepted values and evaluation; no trial writes into caller-owned state.
#[derive(Clone, Debug)]
pub struct NonlinearResult {
    pub values: Vec<f64>,
    pub evaluation: ResidualEvaluation,
    pub structure: ConstraintStructure,
    pub termination: SolveTermination,
    pub iterations: usize,
    pub trial_steps: usize,
    pub relative_residual: f64,
    /// Unsatisfied sources, descending severity, with input index as a tie break.
    pub conflicts: Vec<usize>,
    /// Most recent rejected numerical/domain trial, if any; success may follow it.
    pub rejected_trial: Option<SolveError>,
}

/// Invalid policy, residual contract, or unrepresentable numerical arithmetic.
#[derive(Clone, Debug, PartialEq)]
pub enum SolveError {
    Options(String),
    Evaluation(EvaluationError),
    LinearAlgebra(LinearAlgebraError),
    Arithmetic(String),
}
impl fmt::Display for SolveError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Options(message) | Self::Arithmetic(message) => message.fmt(output),
            Self::Evaluation(cause) => cause.fmt(output),
            Self::LinearAlgebra(cause) => cause.fmt(output),
        }
    }
}
impl Error for SolveError {}
impl From<EvaluationError> for SolveError {
    fn from(cause: EvaluationError) -> Self {
        Self::Evaluation(cause)
    }
}
impl From<LinearAlgebraError> for SolveError {
    fn from(cause: LinearAlgebraError) -> Self {
        Self::LinearAlgebra(cause)
    }
}

/// Solve from a warm start; poll cancellation before each iteration and trial.
/// Example: `let result = solve_constraints(&values, &blocks, SolveOptions::default(), || false)?;`.
pub fn solve_constraints(
    values: &[f64],
    blocks: &[ResidualBlock<'_>],
    options: SolveOptions,
    mut cancelled: impl FnMut() -> bool,
) -> Result<NonlinearResult, SolveError> {
    iteration::run(values, blocks, options, &mut cancelled)
}

fn checked(value: f64, context: &str) -> Result<f64, SolveError> {
    if !value.is_finite() {
        return Err(SolveError::Arithmetic(format!(
            "{context} value {value:?}: expected representable finite arithmetic"
        )));
    }
    Ok(value)
}
