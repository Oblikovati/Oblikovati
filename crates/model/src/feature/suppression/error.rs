// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    parameter::ParameterError,
    quantity::{Dimension, Quantity},
};
use std::fmt::{Display, Formatter};

/// A condition cannot silently enable an operation after a failed read.
#[derive(Clone, Debug, PartialEq)]
pub enum FeatureSuppressionError {
    Parameter(ParameterError<u64>),
    Threshold {
        parameter: u64,
        threshold: Quantity,
    },
    Dimension {
        parameter: u64,
        found: Dimension,
        expected: Dimension,
    },
}

impl Display for FeatureSuppressionError {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Parameter(error) => Display::fmt(error, formatter),
            Self::Threshold {
                parameter,
                threshold,
            } => write!(
                formatter,
                "suppression parameter {parameter} threshold {threshold:?}: expected a finite numeric working-unit quantity"
            ),
            Self::Dimension {
                parameter,
                found,
                expected,
            } => write!(
                formatter,
                "suppression parameter {parameter} dimension {found:?}: expected {expected:?}"
            ),
        }
    }
}

impl std::error::Error for FeatureSuppressionError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Parameter(error) => Some(error),
            _ => None,
        }
    }
}
