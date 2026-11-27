// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_solve::nonlinear::{PreferenceTermination, solve_with_preference};

struct UnitCircle;
impl ResidualSource for UnitCircle {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![
            variables[0]
                .product(&variables[0])?
                .plus(&variables[1].product(&variables[1])?)?
                .shifted(-1.0)?,
        ])
    }
}

#[test]
fn preference_moves_free_coincident_pair_without_changing_dof() {
    let gap = Gap;
    let result = solve_with_preference(
        &[0.0, 5.0],
        &[block(&gap, &[0, 1])],
        &[],
        &[(0, 0.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    assert_eq!(result.preference, PreferenceTermination::Converged);
    assert_eq!(result.solution.termination, SolveTermination::Converged);
    assert_close(result.solution.values[0], 0.0, 1e-7);
    assert_close(result.solution.values[1], 2.0, 1e-7);
    assert_eq!(result.solution.structure.degrees_of_freedom, 1);
}

#[test]
fn dependent_compatible_preferences_move_together() {
    let gap = Gap;
    let result = solve_with_preference(
        &[0.0, 5.0],
        &[block(&gap, &[0, 1])],
        &[],
        &[(0, 0.0), (1, 2.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    assert_eq!(result.preference, PreferenceTermination::Converged);
    assert_close(result.solution.values[0], 0.0, 1e-7);
    assert_close(result.solution.values[1], 2.0, 1e-7);
    assert_eq!(result.solution.structure.degrees_of_freedom, 1);
}

#[test]
fn conflicting_preferences_remain_explicit_and_preserve_the_constraint() {
    let gap = Gap;
    let result = solve_with_preference(
        &[0.0, 5.0],
        &[block(&gap, &[0, 1])],
        &[],
        &[(0, 0.0), (1, 0.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    assert!(matches!(
        result.preference,
        PreferenceTermination::Unavailable | PreferenceTermination::Stalled
    ));
    assert_eq!(result.solution.termination, SolveTermination::Converged);
    assert_close(result.solution.values[0], -1.0, 1e-7);
    assert_close(result.solution.values[1], 1.0, 1e-7);
    assert_eq!(result.solution.structure.degrees_of_freedom, 1);
}

#[test]
fn unavailable_preference_does_not_change_the_satisfied_constraint() {
    let target = Target {
        target: 3.0,
        coefficient: 1.0,
    };
    let result = solve_with_preference(
        &[0.0],
        &[block(&target, &[0])],
        &[],
        &[(0, 0.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    assert_eq!(result.preference, PreferenceTermination::Unavailable);
    assert_close(result.solution.values[0], 3.0, 1e-7);
    assert_eq!(result.solution.structure.degrees_of_freedom, 0);
}

#[test]
fn zero_projected_step_does_not_claim_the_circle_extremum_is_a_minimum() {
    let circle = UnitCircle;
    let result = solve_with_preference(
        &[1.0, 0.0],
        &[block(&circle, &[0, 1])],
        &[],
        &[(0, -1.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    assert_eq!(result.preference, PreferenceTermination::ZeroProjectedStep);
    assert_eq!(result.solution.values, [1.0, 0.0]);
    assert_eq!(result.solution.structure.degrees_of_freedom, 1);
    let alternative = [-1.0_f64, 0.0];
    let alternative_equations = circle
        .evaluate(&DualNumber::seed(&alternative).unwrap())
        .unwrap();
    assert_eq!(alternative_equations[0].value(), 0.0);
    assert!((alternative[0] + 1.0).abs() < (result.solution.values[0] + 1.0).abs());
}

#[test]
fn zero_projected_step_distinguishes_an_unrelated_free_coordinate() {
    let target = Target {
        target: 3.0,
        coefficient: 1.0,
    };
    let result = solve_with_preference(
        &[3.0, 5.0],
        &[block(&target, &[0])],
        &[],
        &[(0, 0.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    assert_eq!(result.preference, PreferenceTermination::ZeroProjectedStep);
    assert_eq!(result.solution.values, [3.0, 5.0]);
}

#[test]
fn preference_follows_the_local_circle_branch_and_keeps_the_constraint() {
    let circle = UnitCircle;
    let result = solve_with_preference(
        &[0.8, 0.6],
        &[block(&circle, &[0, 1])],
        &[],
        &[(0, 0.6)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    assert_eq!(result.preference, PreferenceTermination::Converged);
    assert_close(result.solution.values[0], 0.6, 1e-7);
    assert_close(result.solution.values[1], 0.8, 1e-7);
    assert_eq!(result.solution.structure.degrees_of_freedom, 1);
}

#[test]
fn fixed_coordinates_and_cancellation_are_preserved() {
    let gap = Gap;
    let fixed = solve_with_preference(
        &[0.0, 5.0],
        &[block(&gap, &[0, 1])],
        &[(1, 5.0)],
        &[(0, 0.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    assert_eq!(fixed.preference, PreferenceTermination::Unavailable);
    assert_close(fixed.solution.values[0], 3.0, 1e-7);
    assert_eq!(fixed.solution.values[1], 5.0);
    let cancelled = solve_with_preference(
        &[0.0, 5.0],
        &[block(&gap, &[0, 1])],
        &[],
        &[(0, 0.0)],
        SolveOptions::default(),
        || true,
    )
    .unwrap();
    assert_eq!(cancelled.preference, PreferenceTermination::Cancelled);
    assert_eq!(cancelled.solution.values, [0.0, 5.0]);
}

#[test]
fn empty_preference_is_complete_even_without_iteration_budget() {
    let options = SolveOptions {
        max_iterations: 0,
        ..SolveOptions::default()
    };
    let result = solve_with_preference(&[1.0], &[], &[], &[], options, || false).unwrap();
    assert_eq!(result.preference, PreferenceTermination::Converged);
}

#[test]
fn invalid_preferences_and_unsatisfied_constraints_are_distinct() {
    let options = SolveOptions::default();
    assert!(
        solve_with_preference(&[1.0], &[], &[], &[(0, 1.0), (0, 2.0)], options, || false).is_err()
    );
    assert!(solve_with_preference(&[1.0], &[], &[], &[(0, f64::NAN)], options, || false).is_err());
    assert!(solve_with_preference(&[1.0], &[], &[], &[(1, 0.0)], options, || false).is_err());
    let a = Target {
        target: 1.0,
        coefficient: 1.0,
    };
    let b = Target {
        target: 3.0,
        coefficient: 1.0,
    };
    let result = solve_with_preference(
        &[2.0],
        &[block(&a, &[0]), block(&b, &[0])],
        &[],
        &[(0, 0.0)],
        options,
        || false,
    )
    .unwrap();
    assert_eq!(result.preference, PreferenceTermination::ConstraintFailed);
}

#[test]
fn zero_tangent_preference_can_be_a_farthest_point_on_a_circle() {
    let circle = UnitCircle;
    let result = solve_with_preference(
        &[1.0, 0.0],
        &[block(&circle, &[0, 1])],
        &[],
        &[(0, -1.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    if result.preference == PreferenceTermination::Converged {
        assert_close(result.solution.values[0], -1.0, 1e-7);
        assert_close(result.solution.values[1], 0.0, 1e-7);
    }
}

struct SquaredHeight;
impl ResidualSource for SquaredHeight {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[0].product(&variables[0])?])
    }
}

#[test]
fn singular_constraint_does_not_prove_a_requested_motion() {
    let height = SquaredHeight;
    let result = solve_with_preference(
        &[0.0],
        &[block(&height, &[0])],
        &[],
        &[(0, 1.0)],
        SolveOptions::default(),
        || false,
    )
    .unwrap();
    if result.preference == PreferenceTermination::Converged {
        assert_close(result.solution.values[0], 1.0, 1e-7);
        assert_close(result.solution.values[0].powi(2), 0.0, 1e-7);
    }
}

struct BranchedLinear;
impl ResidualSource for BranchedLinear {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        let x = &variables[0];
        Ok(vec![if x.value() <= 0.0 {
            x.clone()
        } else {
            x.shifted(-1.0)?
        }])
    }
}

#[test]
fn a_sampled_jacobian_does_not_certify_global_affinity() {
    let source = BranchedLinear;
    let blocks = [block(&source, &[0])];
    let left = oblikovati_solve::residual::evaluate_residuals(&[-1.0], &blocks).unwrap();
    let right = oblikovati_solve::residual::evaluate_residuals(&[1.0], &blocks).unwrap();
    assert_eq!(left.jacobian(), right.jacobian());
    assert_ne!(
        right.values()[0],
        left.values()[0] + 2.0 * left.jacobian()[0][0]
    );
}
