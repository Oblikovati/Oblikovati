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

struct MixedAngleOffset;
impl ResidualSource for MixedAngleOffset {
    fn equation_tolerance(&self, row: usize) -> ResidualTolerance {
        if row == 0 {
            ResidualTolerance::Unitless
        } else {
            ResidualTolerance::ModelRelative
        }
    }
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[0].clone(), variables[1].clone()])
    }
}
fn mixed_result(angle: f64, offset: f64) -> NonlinearResult {
    let source = MixedAngleOffset;
    let blocks = [ResidualBlock {
        source: &source,
        variables: &[0, 1],
        equations: 2,
    }];
    let options = SolveOptions {
        max_iterations: 0,
        ..Default::default()
    };
    solve_constraints(&[angle, offset, 1e12], &blocks, options, || false).unwrap()
}
#[test]
fn mixed_source_reports_angular_error_without_relaxing_it_by_length() {
    let result = mixed_result(0.1, 1.0);
    assert_eq!(result.termination, SolveTermination::IterationLimit);
    assert_eq!(result.relative_residual, 0.1);
    assert_eq!(result.conflicts, [0]);
}
#[test]
fn mixed_source_keeps_the_offset_model_relative_and_the_angle_unitless() {
    let result = mixed_result(0.0, 1.0);
    assert_eq!(result.termination, SolveTermination::Converged);
    assert_eq!(result.relative_residual, 1e-12);
    assert!(result.conflicts.is_empty());
}

// #3788: a four-million-unit line couples a small coordinate derivative to a radian column.
struct CoupledCoordinateAngle;
impl ResidualSource for CoupledCoordinateAngle {
    fn tolerance_kind(&self) -> ResidualTolerance {
        ResidualTolerance::Unitless
    }
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![
            variables[0].scaled(2.5e-7)?.plus(&variables[1])?,
            variables[1].clone(),
        ])
    }
}
#[test]
fn mixed_coordinate_and_angle_columns_converge_with_default_iteration_budget() {
    let result = coupled_coordinate_angle_result();
    assert_eq!(
        result.termination,
        SolveTermination::Converged,
        "{result:?}"
    );
    let tolerance = SolveOptions::default().relative_tolerance;
    assert!(
        result
            .evaluation
            .values()
            .iter()
            .all(|value| value.abs() <= tolerance)
    );
}

fn coupled_coordinate_angle_result() -> NonlinearResult {
    let source = CoupledCoordinateAngle;
    let blocks = [ResidualBlock {
        source: &source,
        variables: &[0, 1],
        equations: 2,
    }];
    solve(&[4e6, 0.0], &blocks)
}
