// SPDX-License-Identifier: GPL-2.0-only

use super::{Dimension, Quantity};
use std::fmt;

/// A rejected quantity operation with its original operands or categories.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum QuantityError {
    Incompatible {
        operator: &'static str,
        left: Dimension,
        right: Dimension,
    },
    ZeroDivisor {
        dividend: Quantity,
        divisor: Quantity,
    },
}

impl fmt::Display for QuantityError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Incompatible {
                operator,
                left,
                right,
            } => write!(
                formatter,
                "cannot apply {operator:?} to {left:?} and {right:?}: expected compatible categories and a named result dimension"
            ),
            Self::ZeroDivisor { dividend, divisor } => write!(
                formatter,
                "cannot divide {dividend:?} by {divisor:?}: expected a nonzero divisor"
            ),
        }
    }
}

impl std::error::Error for QuantityError {}
