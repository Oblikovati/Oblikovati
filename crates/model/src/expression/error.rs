// SPDX-License-Identifier: GPL-2.0-only

use crate::quantity::{Quantity, QuantityError};
use std::fmt;

/// Positioned syntax or evaluation failure, ready for parameter health reporting.
#[derive(Clone, Debug, PartialEq)]
pub enum ExpressionError {
    InputContext {
        found: String,
        expected: &'static str,
    },
    Syntax {
        position: usize,
        found: String,
        expected: &'static str,
    },
    Reference {
        name: String,
        expected: &'static str,
    },
    Function {
        name: String,
        found: String,
        expected: &'static str,
    },
    Arithmetic(QuantityError),
    NonFinite {
        source: String,
        value: Quantity,
    },
}

impl fmt::Display for ExpressionError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::InputContext { found, expected } => write!(
                formatter,
                "scalar expression context {found}: expected {expected}"
            ),
            Self::Syntax {
                position,
                found,
                expected,
            } => write!(
                formatter,
                "expression at byte {position}: found {found:?}; expected {expected}"
            ),
            Self::Reference { name, expected } => {
                write!(formatter, "reference {name:?}: expected {expected}")
            }
            Self::Function {
                name,
                found,
                expected,
            } => write!(
                formatter,
                "function {name:?}: found {found}; expected {expected}"
            ),
            Self::Arithmetic(error) => error.fmt(formatter),
            Self::NonFinite { source, value } => write!(
                formatter,
                "expression {source:?} produced {value:?}: expected a finite numeric result"
            ),
        }
    }
}

impl std::error::Error for ExpressionError {}

impl From<QuantityError> for ExpressionError {
    fn from(error: QuantityError) -> Self {
        Self::Arithmetic(error)
    }
}
