// SPDX-License-Identifier: GPL-2.0-only

//! Residual values and Jacobian rows from one immutable automatic-differentiation pass.
mod error;
use crate::ad::DualNumber;
pub use error::{EvaluationError, ResidualError};
use std::ops::Range;

/// A constraint's equations, expressed with the shared derivative primitives.
pub trait ResidualSource {
    /// Evaluate local variables in declared order, returning the declared number of equations.
    /// Example: `Ok(vec![variables[1].minus(&variables[0])?.shifted(-2.0)?])`.
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError>;
}

/// One source's local-to-global variable mapping and stable equation count.
pub struct ResidualBlock<'a> {
    pub source: &'a dyn ResidualSource,
    pub variables: &'a [usize],
    pub equations: usize,
}

/// Owned evaluation in source/equation order; each row has one column per global variable.
#[derive(Clone, Debug, PartialEq)]
pub struct ResidualEvaluation {
    values: Vec<f64>,
    jacobian: Vec<Vec<f64>>,
    source_rows: Vec<Range<usize>>,
    variables: usize,
}
impl ResidualEvaluation {
    /// Inspect equation values; example: `evaluation.values()`.
    pub fn values(&self) -> &[f64] {
        &self.values
    }
    /// Inspect partial derivatives in global variable order; example: `evaluation.jacobian()`.
    pub fn jacobian(&self) -> &[Vec<f64>] {
        &self.jacobian
    }
    /// Map diagnostics back to source blocks, including empty sources; example: `evaluation.source_rows()[0].clone()`.
    pub fn source_rows(&self) -> &[Range<usize>] {
        &self.source_rows
    }
    /// Retain the column count even for an empty equation system; example: `evaluation.variable_count()`.
    pub fn variable_count(&self) -> usize {
        self.variables
    }
}

/// Evaluate all sources once without writing or perturbing input values.
/// Example: `let evaluation = evaluate_residuals(&[1.5, 4.0], &blocks)?;`.
pub fn evaluate_residuals(
    values: &[f64],
    blocks: &[ResidualBlock<'_>],
) -> Result<ResidualEvaluation, EvaluationError> {
    validate_indices(values.len(), blocks)?;
    let seeded = DualNumber::seed(values).map_err(EvaluationError::Input)?;
    let mut result = ResidualEvaluation {
        values: vec![],
        jacobian: vec![],
        source_rows: vec![],
        variables: values.len(),
    };
    for (index, block) in blocks.iter().enumerate() {
        evaluate_block(&seeded, block, index, &mut result)?;
    }
    Ok(result)
}

fn validate_indices(count: usize, blocks: &[ResidualBlock<'_>]) -> Result<(), EvaluationError> {
    for (block, source) in blocks.iter().enumerate() {
        if let Some((local, &global)) = source
            .variables
            .iter()
            .enumerate()
            .find(|(_, global)| **global >= count)
        {
            return Err(EvaluationError::Variable {
                block,
                local,
                global,
                count,
            });
        }
    }
    Ok(())
}

fn evaluate_block(
    seeded: &[DualNumber],
    block: &ResidualBlock<'_>,
    index: usize,
    result: &mut ResidualEvaluation,
) -> Result<(), EvaluationError> {
    let local: Vec<DualNumber> = block
        .variables
        .iter()
        .map(|&global| seeded[global].clone())
        .collect();
    let equations = block
        .source
        .evaluate(&local)
        .map_err(|cause| EvaluationError::Source {
            block: index,
            cause,
        })?;
    if equations.len() != block.equations {
        return Err(EvaluationError::Rows {
            block: index,
            expected: block.equations,
            actual: equations.len(),
        });
    }
    let start = result.values.len();
    for (row, value) in equations.iter().enumerate() {
        append_equation(result, value, index, row)?;
    }
    result.source_rows.push(start..result.values.len());
    Ok(())
}

fn append_equation(
    result: &mut ResidualEvaluation,
    value: &DualNumber,
    block: usize,
    row: usize,
) -> Result<(), EvaluationError> {
    let gradient = value.gradient();
    if !gradient.is_empty() && gradient.len() != result.variables {
        return Err(EvaluationError::Columns {
            block,
            row,
            expected: result.variables,
            actual: gradient.len(),
        });
    }
    result.values.push(value.value());
    result.jacobian.push(if gradient.is_empty() {
        vec![0.0; result.variables]
    } else {
        gradient.to_vec()
    });
    Ok(())
}
