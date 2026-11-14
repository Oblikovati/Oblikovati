// SPDX-License-Identifier: GPL-2.0-only
//! Preference minimization on the local manifold of authored constraints.
use super::{NonlinearResult, SolveError, SolveOptions, SolveTermination, iteration};
use crate::{
    linalg::{ColumnPivotedQr, DenseMatrix},
    residual::ResidualBlock,
};
use std::collections::BTreeSet;

/// Status of a soft preference, separate from satisfaction of authored constraints.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PreferenceTermination {
    Converged,
    Unavailable,
    Stalled,
    IterationLimit,
    Cancelled,
    ConstraintFailed,
}

/// The accepted constraint solution and the separate preference status.
#[derive(Clone, Debug)]
pub struct PreferenceResult {
    pub solution: NonlinearResult,
    pub preference: PreferenceTermination,
}

/// Solve constraints and move toward coordinate preferences without adding authored equations.
/// Example: `solve_with_preference(&values, &blocks, &[], &[(0, 1.0)], options, || false)?`.
pub fn solve_with_preference(
    values: &[f64],
    blocks: &[ResidualBlock<'_>],
    constants: &[(usize, f64)],
    preferences: &[(usize, f64)],
    options: SolveOptions,
    mut cancelled: impl FnMut() -> bool,
) -> Result<PreferenceResult, SolveError> {
    validate_preferences(values.len(), preferences)?;
    let solution = iteration::run(values, blocks, constants, options, &mut cancelled)?;
    if solution.termination != SolveTermination::Converged {
        let preference = if solution.termination == SolveTermination::Cancelled {
            PreferenceTermination::Cancelled
        } else {
            PreferenceTermination::ConstraintFailed
        };
        return Ok(PreferenceResult {
            solution,
            preference,
        });
    }
    optimize(
        solution,
        blocks,
        constants,
        preferences,
        options,
        &mut cancelled,
    )
}

fn validate_preferences(count: usize, preferences: &[(usize, f64)]) -> Result<(), SolveError> {
    let mut seen = BTreeSet::new();
    for &(index, target) in preferences {
        if index >= count || !target.is_finite() || !seen.insert(index) {
            return Err(SolveError::Options(format!(
                "preference ({index}, {target:?}): expected a unique in-range coordinate and finite target"
            )));
        }
    }
    Ok(())
}

fn optimize(
    mut solution: NonlinearResult,
    blocks: &[ResidualBlock<'_>],
    constants: &[(usize, f64)],
    preferences: &[(usize, f64)],
    options: SolveOptions,
    cancelled: &mut impl FnMut() -> bool,
) -> Result<PreferenceResult, SolveError> {
    for step in 0..=options.max_iterations {
        if cancelled() {
            return Ok(PreferenceResult {
                solution,
                preference: PreferenceTermination::Cancelled,
            });
        }
        let (pending, distance) =
            pending_preferences(&solution.values, preferences, options.relative_tolerance);
        if pending.is_empty() {
            return Ok(PreferenceResult {
                solution,
                preference: PreferenceTermination::Converged,
            });
        }
        if step == options.max_iterations {
            break;
        }
        let Some(seed) = preference_seed(&solution, constants, &pending, options.rank_tolerance)?
        else {
            return Ok(PreferenceResult {
                solution,
                preference: PreferenceTermination::Unavailable,
            });
        };
        let next = iteration::run(&seed, blocks, constants, options, cancelled)?;
        if next.termination == SolveTermination::Cancelled {
            return Ok(PreferenceResult {
                solution,
                preference: PreferenceTermination::Cancelled,
            });
        }
        let (_, next_distance) =
            pending_preferences(&next.values, preferences, options.relative_tolerance);
        if next.termination != SolveTermination::Converged || next_distance >= distance {
            return Ok(PreferenceResult {
                solution,
                preference: PreferenceTermination::Stalled,
            });
        }
        solution = next;
    }
    Ok(PreferenceResult {
        solution,
        preference: PreferenceTermination::IterationLimit,
    })
}

fn pending_preferences(
    values: &[f64],
    preferences: &[(usize, f64)],
    tolerance: f64,
) -> (Vec<(usize, f64)>, f64) {
    let mut pending = Vec::new();
    let mut distance = 0.0_f64;
    let mut incomplete = false;
    for &(index, target) in preferences {
        let difference = target - values[index];
        distance = distance.hypot(difference);
        if difference.abs() > tolerance * target.abs().max(values[index].abs()).max(1.0) {
            incomplete = true;
        }
        pending.push((index, difference));
    }
    if !incomplete {
        pending.clear();
    }
    (pending, distance)
}

fn preference_seed(
    solution: &NonlinearResult,
    constants: &[(usize, f64)],
    pending: &[(usize, f64)],
    tolerance: f64,
) -> Result<Option<Vec<f64>>, SolveError> {
    let n = solution.values.len();
    let mut constraints: Vec<Vec<f64>> = solution
        .evaluation
        .jacobian()
        .iter()
        .map(|row| crate::row_scaling::RowScale::new(row).row(row))
        .collect();
    for &(index, _) in constants {
        let mut row = vec![0.0; n];
        row[index] = 1.0;
        constraints.push(row);
    }
    let qr = ColumnPivotedQr::factor(&DenseMatrix::new(n, constraints)?.transpose(), tolerance)?;
    let mut rows = vec![vec![0.0; pending.len()]; n];
    for (column, &(index, _)) in pending.iter().enumerate() {
        rows[index][column] = 1.0;
    }
    let targets: Vec<f64> = pending.iter().map(|&(_, difference)| difference).collect();
    let motion =
        qr.best_fit_free_motion(&DenseMatrix::new(pending.len(), rows)?, &targets, tolerance)?;
    if motion.iter().all(|&value| value == 0.0) {
        return Ok(None);
    }
    let seed = solution
        .values
        .iter()
        .zip(motion)
        .map(|(&value, delta)| super::checked(value + delta, "preference seed"))
        .collect::<Result<_, _>>()?;
    Ok(Some(seed))
}
