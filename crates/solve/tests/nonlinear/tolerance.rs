// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_solve::residual::ResidualTolerance;
struct RadianTarget;
impl ResidualSource for RadianTarget {
    fn tolerance_kind(&self) -> ResidualTolerance {
        ResidualTolerance::Unitless
    }
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[0].clone()])
    }
}
#[test]
fn unitless_error_does_not_converge_early_beside_a_large_length_variable() {
    let angular = RadianTarget;
    let length = Target {
        target: 1e12,
        coefficient: 1.0,
    };
    let blocks = [block(&angular, &[0]), block(&length, &[1])];
    let result = solve(&[0.9, 1e12], &blocks);
    assert_eq!(result.termination, SolveTermination::Converged);
    assert!(result.values[0].abs() <= SolveOptions::default().relative_tolerance);
    assert_eq!(result.values[1], 1e12);
    assert!(result.conflicts.is_empty());
    assert_eq!(length.tolerance_kind(), ResidualTolerance::ModelRelative);
}
#[test]
fn unsatisfied_unitless_source_is_reported_even_when_model_relative_error_is_small() {
    let angular = RadianTarget;
    let options = SolveOptions {
        max_iterations: 0,
        ..Default::default()
    };
    let result =
        solve_constraints(&[1e12, 0.1], &[block(&angular, &[1])], options, || false).unwrap();
    assert_eq!(result.termination, SolveTermination::IterationLimit);
    assert_eq!(result.conflicts, [0]);
    assert_eq!(result.relative_residual, 0.1);
}
