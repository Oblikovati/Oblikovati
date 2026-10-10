// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_solve::{
    ad::DualNumber,
    diagnostics::ConstraintStatus,
    nonlinear::{NonlinearResult, SolveError, SolveOptions, SolveTermination, solve_constraints},
    residual::{ResidualBlock, ResidualError, ResidualSource, evaluate_residuals},
};
use std::cell::Cell;

struct Target {
    target: f64,
    coefficient: f64,
}
impl ResidualSource for Target {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![
            variables[0]
                .shifted(-self.target)?
                .scaled(self.coefficient)?,
        ])
    }
}
struct SquaredTarget(f64);
impl ResidualSource for SquaredTarget {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[0].product(&variables[0])?.shifted(-self.0)?])
    }
}
struct Gap;
impl ResidualSource for Gap {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[1].minus(&variables[0])?.shifted(-2.0)?])
    }
}
struct Constant(f64);
impl ResidualSource for Constant {
    fn evaluate(&self, _: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![DualNumber::constant(self.0)?])
    }
}
struct SquareRoot {
    invalid_trials: Cell<usize>,
}
impl ResidualSource for SquareRoot {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        if variables[0].value() < 0.0 {
            self.invalid_trials.set(self.invalid_trials.get() + 1);
        }
        Ok(vec![variables[0].sqrt()?.shifted(-0.1)?])
    }
}
fn block<'a>(source: &'a dyn ResidualSource, variables: &'a [usize]) -> ResidualBlock<'a> {
    ResidualBlock {
        source,
        variables,
        equations: 1,
    }
}
fn solve(values: &[f64], blocks: &[ResidualBlock<'_>]) -> NonlinearResult {
    solve_constraints(values, blocks, SolveOptions::default(), || false).unwrap()
}
fn assert_close(actual: f64, expected: f64, tolerance: f64) {
    assert!(
        (actual - expected).abs() <= tolerance,
        "actual {actual:?}, expected {expected:?}"
    );
}

#[test]
fn nonlinear_root_retains_final_derivatives_and_deterministic_owned_values() {
    let source = SquaredTarget(4.0);
    let blocks = [block(&source, &[0])];
    let values = [3.0];
    let result = solve(&values, &blocks);
    assert_eq!(result.termination, SolveTermination::Converged);
    assert_close(result.values[0], 2.0, 1e-9);
    assert_eq!(result.structure.status, ConstraintStatus::WellConstrained);
    assert_eq!(
        result.evaluation,
        evaluate_residuals(&result.values, &blocks).unwrap()
    );
    assert!(result.conflicts.is_empty());
    assert_eq!(values, [3.0]);
    let repeat = solve(&values, &blocks);
    assert_eq!(result.values, repeat.values);
    assert_eq!(result.evaluation, repeat.evaluation);
    assert_eq!(result.trial_steps, repeat.trial_steps);
    assert!(result.iterations > 0 && result.iterations < 100);
}

#[test]
fn convergence_is_independent_from_under_and_over_constraint_structure() {
    let gap = Gap;
    let source = block(&gap, &[0, 1]);
    let result = solve(&[0.0, 5.0], &[source]);
    assert_eq!(result.termination, SolveTermination::Converged);
    assert_close(result.values[0], 1.5, 1e-8);
    assert_close(result.values[1], 3.5, 1e-8);
    assert_eq!(result.structure.degrees_of_freedom, 1);
    let result = solve(&[0.0, 5.0], &[block(&gap, &[0, 1]), block(&gap, &[0, 1])]);
    assert_eq!(result.termination, SolveTermination::Converged);
    assert_eq!(result.structure.status, ConstraintStatus::OverConstrained);
    assert_eq!(result.structure.degrees_of_freedom, 1);
    assert!(result.conflicts.is_empty());
}

#[test]
fn inconsistent_targets_stop_with_ordered_unsatisfied_sources() {
    let first = Target {
        target: 1.0,
        coefficient: 1.0,
    };
    let second = Target {
        target: 3.0,
        coefficient: 1.0,
    };
    let blocks = [block(&first, &[0]), block(&second, &[0])];
    let result = solve(&[2.0], &blocks);
    assert_eq!(result.termination, SolveTermination::Stalled);
    assert_eq!(result.structure.status, ConstraintStatus::OverConstrained);
    assert_eq!(result.conflicts, [0, 1]);
    assert_eq!(result.values, [2.0]);
    assert!(result.trial_steps <= SolveOptions::default().max_trial_steps);
    let severe = Constant(8.0);
    let result = solve(&[], &[block(&first_constant(), &[]), block(&severe, &[])]);
    assert_eq!(result.conflicts, [1, 0]);
}
fn first_constant() -> Constant {
    Constant(2.0)
}

#[test]
fn empty_and_zero_variable_systems_do_not_hide_unsatisfied_equations() {
    let empty = solve(&[], &[]);
    assert_eq!(empty.termination, SolveTermination::Converged);
    assert_eq!(empty.structure.status, ConstraintStatus::WellConstrained);
    let free = solve(&[9.0, 2.0], &[]);
    assert_eq!(free.termination, SolveTermination::Converged);
    assert_eq!(free.values, [9.0, 2.0]);
    assert_eq!(free.structure.degrees_of_freedom, 2);
    let zero = Constant(0.0);
    assert_eq!(
        solve(&[], &[block(&zero, &[])]).termination,
        SolveTermination::Converged
    );
    let bad = Constant(1.0);
    let result = solve(&[], &[block(&bad, &[])]);
    assert_eq!(result.termination, SolveTermination::Stalled);
    assert_eq!(result.conflicts, [0]);
    let square = SquaredTarget(4.0);
    let singular = solve(&[0.0], &[block(&square, &[0])]);
    assert_eq!(singular.termination, SolveTermination::Stalled);
    assert_eq!(singular.values, [0.0]);
    assert_eq!(singular.structure.rank, 0);
    assert_eq!(singular.conflicts, [0]);
}

#[test]
fn rejected_domain_trials_are_bounded_and_do_not_replace_final_valid_state() {
    let source = SquareRoot {
        invalid_trials: Cell::new(0),
    };
    let blocks = [block(&source, &[0])];
    let result = solve(&[4.0], &blocks);
    assert_eq!(result.termination, SolveTermination::Converged);
    assert_close(result.values[0], 0.01, 1e-8);
    assert!(source.invalid_trials.get() > 0);
    assert!(matches!(
        result.rejected_trial,
        Some(SolveError::Evaluation(_))
    ));
    assert_eq!(
        result.evaluation,
        evaluate_residuals(&result.values, &blocks).unwrap()
    );
    let options = SolveOptions {
        max_trial_steps: 1,
        ..SolveOptions::default()
    };
    let limited = solve_constraints(&[4.0], &blocks, options, || false).unwrap();
    assert_eq!(limited.termination, SolveTermination::Stalled);
    assert_eq!(limited.values, [4.0]);
    assert_eq!(limited.trial_steps, 1);
}

#[test]
fn iteration_limits_and_injected_cancellation_retain_last_accepted_evaluation() {
    let source = SquaredTarget(4.0);
    let blocks = [block(&source, &[0])];
    let options = SolveOptions {
        max_iterations: 0,
        ..SolveOptions::default()
    };
    let limited = solve_constraints(&[3.0], &blocks, options, || false).unwrap();
    assert_eq!(limited.termination, SolveTermination::IterationLimit);
    assert_eq!(limited.iterations, 0);
    let options = SolveOptions {
        max_iterations: 1,
        ..options
    };
    let result = solve_constraints(&[3.0], &blocks, options, || false).unwrap();
    assert_eq!(result.termination, SolveTermination::IterationLimit);
    assert_eq!(result.iterations, 1);
    assert_eq!(
        result.evaluation,
        evaluate_residuals(&result.values, &blocks).unwrap()
    );
    let mut polls = 0;
    let cancelled = solve_constraints(&[3.0], &blocks, SolveOptions::default(), || {
        polls += 1;
        polls == 2
    })
    .unwrap();
    assert_eq!(cancelled.termination, SolveTermination::Cancelled);
    assert_eq!(cancelled.values, [3.0]);
    assert_eq!(cancelled.trial_steps, 0);
    let cancelled = solve_constraints(&[3.0], &blocks, SolveOptions::default(), || true).unwrap();
    assert_eq!(cancelled.termination, SolveTermination::Cancelled);
}

#[test]
fn coordinate_and_equation_unit_sweeps_preserve_roots() {
    for scale in [1e-100, 1e-6, 1.0, 1e6, 1e100] {
        for coefficient in [1e-200, 1.0, 1e200] {
            let source = Target {
                target: 2.0 * scale,
                coefficient,
            };
            let result = solve(&[3.0 * scale], &[block(&source, &[0])]);
            assert_eq!(
                result.termination,
                SolveTermination::Converged,
                "scale {scale}, coefficient {coefficient}"
            );
            assert_close(result.values[0] / scale, 2.0, 1e-8);
            assert_eq!(result.structure.rank, 1);
        }
    }
}

#[test]
fn invalid_policy_and_initial_evaluation_fail_before_iteration() {
    for tolerance in [0.0, -1.0, 1.0, f64::NAN, f64::INFINITY] {
        let options = SolveOptions {
            relative_tolerance: tolerance,
            ..SolveOptions::default()
        };
        assert!(matches!(
            solve_constraints(&[], &[], options, || false),
            Err(SolveError::Options(_))
        ));
        let options = SolveOptions {
            rank_tolerance: tolerance,
            ..SolveOptions::default()
        };
        assert!(matches!(
            solve_constraints(&[], &[], options, || false),
            Err(SolveError::Options(_))
        ));
    }
    let options = SolveOptions {
        max_trial_steps: 0,
        ..SolveOptions::default()
    };
    assert!(
        solve_constraints(&[], &[], options, || false)
            .unwrap_err()
            .to_string()
            .contains("max_trial_steps 0")
    );
    assert!(matches!(
        solve_constraints(&[f64::NAN], &[], SolveOptions::default(), || false),
        Err(SolveError::Evaluation(_))
    ));
    let source = SquareRoot {
        invalid_trials: Cell::new(0),
    };
    assert!(matches!(
        solve_constraints(
            &[-1.0],
            &[block(&source, &[0])],
            SolveOptions::default(),
            || false
        ),
        Err(SolveError::Evaluation(_))
    ));
}

#[test]
fn solutions_and_structure_match_captured_go_reference() {
    let square = SquaredTarget(4.0);
    let gap = Gap;
    let results = [
        solve(&[3.0], &[block(&square, &[0])]),
        solve(&[0.0, 5.0], &[block(&gap, &[0, 1])]),
    ];
    for (result, fixture) in results
        .iter()
        .zip(include_str!("fixtures/go-nonlinear.tsv").lines())
    {
        let columns: Vec<&str> = fixture.split('\t').collect();
        for (&actual, expected) in result.values.iter().zip(columns[1].split(',')) {
            assert_close(actual, expected.parse().unwrap(), 1e-8);
        }
        assert_eq!(
            result.termination == SolveTermination::Converged,
            columns[2] == "true"
        );
        assert_eq!(
            result.structure.degrees_of_freedom,
            columns[3].parse::<usize>().unwrap()
        );
        assert_eq!(
            result.structure.redundant_equations,
            columns[4].parse::<usize>().unwrap()
        );
    }
}

struct Distance(f64);
impl ResidualSource for Distance {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        let dx = variables[2].minus(&variables[0])?;
        let dy = variables[3].minus(&variables[1])?;
        Ok(vec![dx.hypot(&dy)?.shifted(-self.0)?])
    }
}
struct Radius(f64);
impl ResidualSource for Radius {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[0].hypot(&variables[1])?.shifted(-self.0)?])
    }
}
#[test]
fn go_mixed_length_regression_converges_without_special_cases() {
    let radius = Radius(1000.0);
    let distance = Distance(0.01);
    let blocks = [block(&radius, &[0, 1]), block(&distance, &[0, 1, 2, 3])];
    let result = solve(&[900.0, 100.0, 900.004, 100.002], &blocks);
    assert_eq!(result.termination, SolveTermination::Converged);
    assert_close(result.values[0].hypot(result.values[1]), 1000.0, 1e-7);
    assert_close(
        (result.values[2] - result.values[0]).hypot(result.values[3] - result.values[1]),
        0.01,
        1e-7,
    );
    assert_eq!(result.structure.degrees_of_freedom, 2);
}

struct ChangingShape;
impl ResidualSource for ChangingShape {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        if variables[0].value() > 0.5 {
            return Ok(vec![]);
        }
        Ok(vec![variables[0].shifted(-1.0)?])
    }
}
#[test]
fn trial_contract_violations_are_errors_instead_of_damping_fallbacks() {
    let source = ChangingShape;
    let error = solve_constraints(
        &[0.0],
        &[block(&source, &[0])],
        SolveOptions::default(),
        || false,
    )
    .unwrap_err();
    assert!(matches!(error, SolveError::Evaluation(_)));
    assert!(
        error
            .to_string()
            .contains("returned 0 equations: expected 1")
    );
}
