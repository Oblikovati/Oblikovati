// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_solve::{
    ad::DualNumber,
    diagnostics::{ConstraintStatus, analyze_structure, redundant_sources},
    residual::{ResidualBlock, ResidualError, ResidualSource, evaluate_residuals},
};
use std::cell::Cell;

struct LinearRows {
    coefficients: Vec<Vec<f64>>,
    calls: Cell<usize>,
}
impl ResidualSource for LinearRows {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        self.calls.set(self.calls.get() + 1);
        self.coefficients
            .iter()
            .map(|row| {
                row.iter().zip(variables).try_fold(
                    DualNumber::constant(0.0)?,
                    |sum, (&coefficient, variable)| Ok(sum.plus(&variable.scaled(coefficient)?)?),
                )
            })
            .collect()
    }
}
fn source(rows: &[&[f64]]) -> LinearRows {
    LinearRows {
        coefficients: rows.iter().map(|row| row.to_vec()).collect(),
        calls: Cell::new(0),
    }
}

#[test]
fn independent_and_dependent_equations_report_go_status_precedence() {
    for (rows, rank, free, redundant, status) in [
        (
            vec![vec![1.0, 0.0], vec![0.0, 1.0]],
            2,
            0,
            0,
            ConstraintStatus::WellConstrained,
        ),
        (
            vec![vec![-1.0, 1.0]],
            1,
            1,
            0,
            ConstraintStatus::UnderConstrained,
        ),
        (
            vec![vec![-1.0, 1.0], vec![-1.0, 1.0]],
            1,
            1,
            1,
            ConstraintStatus::OverConstrained,
        ),
    ] {
        let source = LinearRows {
            coefficients: rows,
            calls: Cell::new(0),
        };
        let equations = source.coefficients.len();
        let evaluation = evaluate_residuals(
            &[1.0, 2.0],
            &[ResidualBlock {
                source: &source,
                variables: &[0, 1],
                equations,
            }],
        )
        .unwrap();
        let result = analyze_structure(&evaluation, 1e-9).unwrap();
        assert_eq!(
            (
                result.variables,
                result.equations,
                result.rank,
                result.degrees_of_freedom,
                result.redundant_equations,
                result.status
            ),
            (2, equations, rank, free, redundant, status)
        );
        assert!(status.to_string().ends_with("constrained"));
        assert_eq!(source.calls.get(), 1);
    }
}

#[test]
fn individual_removal_candidates_do_not_include_partly_essential_blocks_or_empty_sources() {
    let both = source(&[&[1.0, 0.0], &[0.0, 1.0]]);
    let duplicate = source(&[&[1.0, 0.0]]);
    let empty = source(&[]);
    let blocks = [
        ResidualBlock {
            source: &both,
            variables: &[0, 1],
            equations: 2,
        },
        ResidualBlock {
            source: &duplicate,
            variables: &[0, 1],
            equations: 1,
        },
        ResidualBlock {
            source: &empty,
            variables: &[],
            equations: 0,
        },
    ];
    let evaluation = evaluate_residuals(&[0.0, 0.0], &blocks).unwrap();
    let saved = evaluation.clone();
    assert_eq!(redundant_sources(&evaluation, 1e-9).unwrap(), [1]);
    assert_eq!(evaluation, saved);
    assert_eq!(
        (both.calls.get(), duplicate.calls.get(), empty.calls.get()),
        (1, 1, 1)
    );
    let pair = [
        ResidualBlock {
            source: &duplicate,
            variables: &[0, 1],
            equations: 1,
        },
        ResidualBlock {
            source: &duplicate,
            variables: &[0, 1],
            equations: 1,
        },
    ];
    let duplicated = evaluate_residuals(&[0.0, 0.0], &pair).unwrap();
    assert_eq!(redundant_sources(&duplicated, 1e-9).unwrap(), [0, 1]);
}

#[test]
fn normalization_handles_tiny_huge_and_mixed_row_scales_without_absolute_cutoffs() {
    for scales in [[1e-300, 1e300], [f64::MAX, f64::MAX], [1.0, 1.0]] {
        let source = source(&[&[scales[0], scales[0]], &[scales[1], -scales[1]]]);
        let evaluation = evaluate_residuals(
            &[0.0, 0.0],
            &[ResidualBlock {
                source: &source,
                variables: &[0, 1],
                equations: 2,
            }],
        )
        .unwrap();
        let result = analyze_structure(&evaluation, 1e-9).unwrap();
        assert_eq!(result.rank, 2);
        assert_eq!(result.degrees_of_freedom, 0);
        assert!(redundant_sources(&evaluation, 1e-9).unwrap().is_empty());
    }
}

#[test]
fn empty_axes_and_zero_rows_remain_structural_not_convergence_claims() {
    let empty = evaluate_residuals(&[1.0, 2.0, 3.0], &[]).unwrap();
    assert_eq!(
        analyze_structure(&empty, 1e-9).unwrap().degrees_of_freedom,
        3
    );
    assert!(redundant_sources(&empty, 1e-9).unwrap().is_empty());
    let no_variables = evaluate_residuals(&[], &[]).unwrap();
    assert_eq!(
        analyze_structure(&no_variables, 1e-9).unwrap().status,
        ConstraintStatus::WellConstrained
    );
    let zero = source(&[&[0.0, 0.0]]);
    let evaluation = evaluate_residuals(
        &[1.0, 2.0],
        &[ResidualBlock {
            source: &zero,
            variables: &[0, 1],
            equations: 1,
        }],
    )
    .unwrap();
    let result = analyze_structure(&evaluation, 1e-9).unwrap();
    assert_eq!(
        (
            result.rank,
            result.degrees_of_freedom,
            result.redundant_equations
        ),
        (0, 2, 1)
    );
    assert_eq!(redundant_sources(&evaluation, 1e-9).unwrap(), [0]);
}

#[test]
fn numerical_tolerance_is_applied_and_invalid_tolerance_is_not_skipped_for_empty_systems() {
    let source = source(&[&[1.0, 1.0], &[1.0, 1.0 + 1e-12]]);
    let evaluation = evaluate_residuals(
        &[0.0, 0.0],
        &[ResidualBlock {
            source: &source,
            variables: &[0, 1],
            equations: 2,
        }],
    )
    .unwrap();
    assert_eq!(analyze_structure(&evaluation, 1e-9).unwrap().rank, 1);
    assert_eq!(analyze_structure(&evaluation, 1e-14).unwrap().rank, 2);
    let empty = evaluate_residuals(&[], &[]).unwrap();
    assert!(analyze_structure(&empty, 0.0).is_err());
    assert!(redundant_sources(&empty, f64::NAN).is_err());
}
