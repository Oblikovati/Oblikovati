// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_solve::{
    ad::DualNumber,
    residual::{EvaluationError, ResidualBlock, ResidualError, ResidualSource, evaluate_residuals},
};
use std::{cell::Cell, error::Error};

struct Gap;
impl ResidualSource for Gap {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[1].minus(&variables[0])?.shifted(-2.0)?])
    }
}
struct SharedVariable;
impl ResidualSource for SharedVariable {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[0].plus(&variables[1].scaled(2.0)?)?])
    }
}
struct Polynomial {
    calls: Cell<usize>,
}
impl ResidualSource for Polynomial {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        self.calls.set(self.calls.get() + 1);
        Ok(vec![
            variables[0].product(&variables[1])?,
            variables[0].product(&variables[0])?,
        ])
    }
}
struct ConstantEquation;
impl ResidualSource for ConstantEquation {
    fn evaluate(&self, _: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![DualNumber::constant(7.0)?])
    }
}
struct EmptySource;
impl ResidualSource for EmptySource {
    fn evaluate(&self, _: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![])
    }
}

#[test]
fn ordered_blocks_map_shared_variables_and_constants_into_one_jacobian() {
    let polynomial = Polynomial {
        calls: Cell::new(0),
    };
    let blocks = [
        ResidualBlock {
            source: &Gap,
            variables: &[2, 0],
            equations: 1,
        },
        ResidualBlock {
            source: &SharedVariable,
            variables: &[1, 1],
            equations: 1,
        },
        ResidualBlock {
            source: &polynomial,
            variables: &[0, 2],
            equations: 2,
        },
        ResidualBlock {
            source: &ConstantEquation,
            variables: &[],
            equations: 1,
        },
        ResidualBlock {
            source: &EmptySource,
            variables: &[],
            equations: 0,
        },
    ];
    let input = [4.0, 3.0, 1.5];
    let result = evaluate_residuals(&input, &blocks).unwrap();
    assert_eq!(result.variable_count(), 3);
    assert_eq!(result.values(), [0.5, 9.0, 6.0, 16.0, 7.0]);
    assert_eq!(
        result.jacobian(),
        [
            vec![1.0, 0.0, -1.0],
            vec![0.0, 3.0, 0.0],
            vec![1.5, 0.0, 4.0],
            vec![8.0, 0.0, 0.0],
            vec![0.0; 3]
        ]
    );
    assert_eq!(result.source_rows(), [0..1, 1..2, 2..4, 4..5, 5..5]);
    assert_eq!(polynomial.calls.get(), 1);
    assert_eq!(input, [4.0, 3.0, 1.5]);
    assert_eq!(evaluate_residuals(&input, &blocks).unwrap(), result);
    assert_eq!(polynomial.calls.get(), 2);
}

#[test]
fn empty_system_and_no_active_variables_keep_explicit_shapes() {
    let empty = evaluate_residuals(&[1.0, 2.0], &[]).unwrap();
    assert!(empty.values().is_empty());
    assert!(empty.jacobian().is_empty());
    assert!(empty.source_rows().is_empty());
    assert_eq!(empty.variable_count(), 2);
    let constant = evaluate_residuals(
        &[],
        &[ResidualBlock {
            source: &ConstantEquation,
            variables: &[],
            equations: 1,
        }],
    )
    .unwrap();
    assert_eq!(constant.values(), [7.0]);
    assert_eq!(constant.jacobian(), [Vec::<f64>::new()]);
}

#[test]
fn invalid_mapping_is_refused_before_any_source_is_called() {
    let polynomial = Polynomial {
        calls: Cell::new(0),
    };
    let blocks = [
        ResidualBlock {
            source: &polynomial,
            variables: &[0, 0],
            equations: 2,
        },
        ResidualBlock {
            source: &Gap,
            variables: &[0, 4],
            equations: 1,
        },
    ];
    let error = evaluate_residuals(&[1.0], &blocks).unwrap_err();
    assert_eq!(
        error,
        EvaluationError::Variable {
            block: 1,
            local: 1,
            global: 4,
            count: 1
        }
    );
    assert!(error.to_string().contains("maps to 4"));
    assert!(error.source().is_none());
    assert_eq!(polynomial.calls.get(), 0);
    let input_error = evaluate_residuals(&[f64::NAN], &[]).unwrap_err();
    assert!(input_error.to_string().contains("NaN"));
    assert!(input_error.source().is_some());
}

struct WrongGradient;
impl ResidualSource for WrongGradient {
    fn evaluate(&self, _: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![DualNumber::variable(3.0, 0, 2)?])
    }
}
struct RefusedDomain;
impl ResidualSource for RefusedDomain {
    fn evaluate(&self, _: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Err(ResidualError::Domain(
            "radius -1: expected a positive radius".into(),
        ))
    }
}
struct InvalidDerivative;
impl ResidualSource for InvalidDerivative {
    fn evaluate(&self, _: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![DualNumber::constant(-1.0)?.sqrt()?])
    }
}

#[test]
fn bad_shape_and_source_failures_identify_the_block_without_partial_success() {
    let wrong_rows = evaluate_residuals(
        &[1.0],
        &[ResidualBlock {
            source: &ConstantEquation,
            variables: &[],
            equations: 2,
        }],
    )
    .unwrap_err();
    assert_eq!(
        wrong_rows,
        EvaluationError::Rows {
            block: 0,
            expected: 2,
            actual: 1
        }
    );
    assert!(wrong_rows.to_string().contains("expected 2"));
    let wrong_columns = evaluate_residuals(
        &[1.0],
        &[ResidualBlock {
            source: &WrongGradient,
            variables: &[],
            equations: 1,
        }],
    )
    .unwrap_err();
    assert_eq!(
        wrong_columns,
        EvaluationError::Columns {
            block: 0,
            row: 0,
            expected: 1,
            actual: 2
        }
    );
    assert!(wrong_columns.to_string().contains("2 partials"));
    for source in [&RefusedDomain as &dyn ResidualSource, &InvalidDerivative] {
        let error = evaluate_residuals(
            &[1.0],
            &[ResidualBlock {
                source,
                variables: &[],
                equations: 1,
            }],
        )
        .unwrap_err();
        assert!(error.to_string().contains("block 0"));
        assert!(error.source().is_some());
        let EvaluationError::Source { cause, .. } = error else {
            panic!("expected source error")
        };
        match cause {
            ResidualError::Domain(_) => assert!(cause.source().is_none()),
            ResidualError::Derivative(_) => assert!(cause.source().is_some()),
        }
    }
}
