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
    Category {
        parameter: u64,
        category: String,
    },
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
    fn fmt(&self, out: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Parameter(error) => Display::fmt(error, out),
            Self::Category {
                parameter,
                category,
            } => suppression_category(out, *parameter, category),
            Self::Threshold {
                parameter,
                threshold,
            } => suppression_threshold(out, *parameter, *threshold),
            Self::Dimension {
                parameter,
                found,
                expected,
            } => suppression_dimension(out, (*parameter, *found, *expected)),
        }
    }
}

fn suppression_category(
    out: &mut Formatter<'_>,
    parameter: u64,
    category: &str,
) -> std::fmt::Result {
    write!(
        out,
        "suppression parameter {parameter} category {category:?}: expected a canonical quantity category"
    )
}

fn suppression_threshold(
    out: &mut Formatter<'_>,
    parameter: u64,
    threshold: Quantity,
) -> std::fmt::Result {
    write!(
        out,
        "suppression parameter {parameter} threshold {threshold:?}: expected a finite numeric working-unit quantity"
    )
}

fn suppression_dimension(
    out: &mut Formatter<'_>,
    (parameter, found, expected): (u64, Dimension, Dimension),
) -> std::fmt::Result {
    write!(
        out,
        "suppression parameter {parameter} dimension {found:?}: expected {expected:?}"
    )
}

impl std::error::Error for FeatureSuppressionError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Parameter(error) => Some(error),
            _ => None,
        }
    }
}
