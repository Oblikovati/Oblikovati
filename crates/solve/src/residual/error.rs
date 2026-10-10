// SPDX-License-Identifier: GPL-2.0-only
use crate::ad::DerivativeError;
use std::{error::Error, fmt};

/// A source can refuse a geometric domain or propagate checked derivative arithmetic.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ResidualError {
    Domain(String),
    Derivative(DerivativeError),
}
impl From<DerivativeError> for ResidualError {
    fn from(error: DerivativeError) -> Self {
        Self::Derivative(error)
    }
}
impl fmt::Display for ResidualError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Domain(reason) => reason.fmt(output),
            Self::Derivative(error) => error.fmt(output),
        }
    }
}
impl Error for ResidualError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Domain(_) => None,
            Self::Derivative(error) => Some(error),
        }
    }
}

/// Failed block/row coordinates and expected shape; no partial evaluation is returned.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum EvaluationError {
    Input(DerivativeError),
    Variable {
        block: usize,
        local: usize,
        global: usize,
        count: usize,
    },
    Source {
        block: usize,
        cause: ResidualError,
    },
    Rows {
        block: usize,
        expected: usize,
        actual: usize,
    },
    Columns {
        block: usize,
        row: usize,
        expected: usize,
        actual: usize,
    },
}
impl fmt::Display for EvaluationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Input(error) => write!(output, "solver input: {error}"),
            Self::Variable {
                block,
                local,
                global,
                count,
            } => write!(
                output,
                "residual block {block} local variable {local} maps to {global}: expected global index below {count}"
            ),
            Self::Source { block, cause } => write!(output, "residual block {block}: {cause}"),
            Self::Rows {
                block,
                expected,
                actual,
            } => write!(
                output,
                "residual block {block} returned {actual} equations: expected {expected}"
            ),
            Self::Columns {
                block,
                row,
                expected,
                actual,
            } => write!(
                output,
                "residual block {block} row {row} returned {actual} partials: expected {expected} global columns or a constant"
            ),
        }
    }
}
impl Error for EvaluationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Input(error) => Some(error),
            Self::Source { cause, .. } => Some(cause),
            _ => None,
        }
    }
}
