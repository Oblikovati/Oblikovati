// SPDX-License-Identifier: GPL-2.0-only

use super::{NonlinearResult, SolveError, SolveOptions, SolveTermination, metrics, step};
use crate::{
    diagnostics::analyze_structure,
    residual::{EvaluationError, ResidualBlock, ResidualEvaluation, evaluate_residuals},
};

struct Iterate {
    values: Vec<f64>,
    evaluation: ResidualEvaluation,
    scaled: metrics::ScaledEvaluation,
    damping: f64,
    iterations: usize,
    trials: usize,
    rejected: Option<SolveError>,
}

pub(super) fn run(
    values: &[f64],
    blocks: &[ResidualBlock<'_>],
    options: SolveOptions,
    cancelled: &mut impl FnMut() -> bool,
) -> Result<NonlinearResult, SolveError> {
    validate_options(options)?;
    let mut state = initial_state(values, blocks)?;
    let termination = iterate(&mut state, blocks, options, cancelled)?;
    finish(state, options, termination)
}

fn initial_state(values: &[f64], blocks: &[ResidualBlock<'_>]) -> Result<Iterate, SolveError> {
    let evaluation = evaluate_residuals(values, blocks)?;
    let scaled = metrics::scale_evaluation(&evaluation, metrics::model_scale(values), blocks)?;
    Ok(Iterate {
        values: values.to_vec(),
        evaluation,
        scaled,
        damping: step::INITIAL_DAMPING,
        iterations: 0,
        trials: 0,
        rejected: None,
    })
}

fn validate_options(options: SolveOptions) -> Result<(), SolveError> {
    if options.max_trial_steps == 0 {
        return Err(SolveError::Options(
            "max_trial_steps 0: expected at least one bounded trial".into(),
        ));
    }
    for (name, value) in [
        ("relative_tolerance", options.relative_tolerance),
        ("rank_tolerance", options.rank_tolerance),
    ] {
        validate_tolerance(name, value)?;
    }

    Ok(())
}

fn validate_tolerance(name: &str, value: f64) -> Result<(), SolveError> {
    if !value.is_finite() || value <= 0.0 || value >= 1.0 {
        return Err(SolveError::Options(format!(
            "{name} {value:?}: expected finite tolerance strictly between 0 and 1"
        )));
    }
    Ok(())
}

fn iterate(
    state: &mut Iterate,
    blocks: &[ResidualBlock<'_>],
    options: SolveOptions,
    cancelled: &mut impl FnMut() -> bool,
) -> Result<SolveTermination, SolveError> {
    loop {
        if cancelled() {
            return Ok(SolveTermination::Cancelled);
        }
        if metrics::infinity_norm(&state.scaled.residuals) <= options.relative_tolerance {
            return Ok(SolveTermination::Converged);
        }
        if state.iterations == options.max_iterations {
            return Ok(SolveTermination::IterationLimit);
        }
        if let Some(reason) = advance(state, blocks, options.max_trial_steps, cancelled)? {
            return Ok(reason);
        }
        state.iterations += 1;
    }
}

fn advance(
    state: &mut Iterate,
    blocks: &[ResidualBlock<'_>],
    limit: usize,
    cancelled: &mut impl FnMut() -> bool,
) -> Result<Option<SolveTermination>, SolveError> {
    let mut growth = 2.0;
    for _ in 0..limit {
        if cancelled() {
            return Ok(Some(SolveTermination::Cancelled));
        }
        if state.damping >= step::MAX_DAMPING {
            break;
        }
        state.trials += 1;
        if attempt(state, blocks)? {
            return Ok(None);
        }
        state.damping *= growth;
        growth *= 2.0;
    }
    Ok(Some(SolveTermination::Stalled))
}

fn attempt(state: &mut Iterate, blocks: &[ResidualBlock<'_>]) -> Result<bool, SolveError> {
    match evaluate_trial(state, blocks) {
        Ok(Some(trial)) => {
            state.values = trial.values;
            state.evaluation = trial.evaluation;
            state.scaled = trial.scaled;
            state.damping = trial.damping;
            Ok(true)
        }
        Ok(None) => Ok(false),
        Err(cause) if retryable(&cause) => {
            state.rejected = Some(cause);
            Ok(false)
        }
        Err(cause) => Err(cause),
    }
}

struct AcceptedTrial {
    values: Vec<f64>,
    evaluation: ResidualEvaluation,
    scaled: metrics::ScaledEvaluation,
    damping: f64,
}
fn evaluate_trial(
    state: &Iterate,
    blocks: &[ResidualBlock<'_>],
) -> Result<Option<AcceptedTrial>, SolveError> {
    let delta = step::damped_delta(&state.scaled, state.values.len(), state.damping)?;
    let values = step::trial_values(&state.values, &delta, state.scaled.model_scale)?;
    let evaluation = evaluate_residuals(&values, blocks)?;
    // Compare both points with the SAME weights; moving derivatives cannot fake a decrease.
    let trial = metrics::scaled_residuals(
        &evaluation,
        &state.scaled.weights,
        state.scaled.model_scale,
        &state.scaled.domains,
    )?;
    let next = metrics::norm(&trial)?;
    if next >= metrics::norm(&state.scaled.residuals)? {
        return Ok(None);
    }
    accept_trial(state, values, evaluation, &delta, next, blocks).map(Some)
}

fn accept_trial(
    state: &Iterate,
    values: Vec<f64>,
    evaluation: ResidualEvaluation,
    delta: &[f64],
    next: f64,
    blocks: &[ResidualBlock<'_>],
) -> Result<AcceptedTrial, SolveError> {
    let damping = step::accepted_damping(&state.scaled, delta, next, state.damping)?;
    let scaled = metrics::scale_evaluation(&evaluation, state.scaled.model_scale, blocks)?;
    Ok(AcceptedTrial {
        values,
        evaluation,
        scaled,
        damping,
    })
}

fn retryable(cause: &SolveError) -> bool {
    matches!(
        cause,
        SolveError::Arithmetic(_)
            | SolveError::LinearAlgebra(_)
            | SolveError::Evaluation(EvaluationError::Source { .. } | EvaluationError::Input(_))
    )
}

fn finish(
    state: Iterate,
    options: SolveOptions,
    termination: SolveTermination,
) -> Result<NonlinearResult, SolveError> {
    let structure = analyze_structure(&state.evaluation, options.rank_tolerance)?;
    let residuals = &state.scaled.residuals;
    let conflicts = metrics::conflicts(&state.evaluation, residuals, options.relative_tolerance);
    Ok(NonlinearResult {
        values: state.values,
        evaluation: state.evaluation,
        structure,
        termination,
        iterations: state.iterations,
        trial_steps: state.trials,
        relative_residual: metrics::infinity_norm(residuals),
        conflicts,
        rejected_trial: state.rejected,
    })
}
