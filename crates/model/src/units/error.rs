// SPDX-License-Identifier: GPL-2.0-only

use crate::quantity::Dimension;
use std::fmt;

/// Invalid authored input or document unit configuration.
#[derive(Clone, Debug, PartialEq)]
pub enum UnitError {
    UnknownUnit(String),
    Display(String),
    WrongCategory {
        name: String,
        expected: Dimension,
    },
    InvalidNumber(String),
    InvalidScale(f64),
    Conversion {
        value: f64,
        from: String,
        to: String,
        expected: &'static str,
    },
}

impl fmt::Display for UnitError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Display(found) => write!(
                formatter,
                "unit display: found {found}; expected a finite numeric value within the selected format limits"
            ),
            Self::UnknownUnit(name) => {
                write!(formatter, "unit {name:?}: expected a registered unit name")
            }
            Self::WrongCategory { name, expected } => write!(
                formatter,
                "unit {name:?}: expected a registered {expected:?} unit"
            ),
            Self::InvalidNumber(input) => write!(
                formatter,
                "quantity {input:?}: expected a finite decimal number followed by an optional unit"
            ),
            Self::Conversion {
                value,
                from,
                to,
                expected,
            } => write!(
                formatter,
                "conversion of {value} from {from:?} to {to:?}: expected {expected}"
            ),
            Self::InvalidScale(scale) => write!(
                formatter,
                "working scale {scale}: expected positive finite centimetres per unit with representable square and cube"
            ),
        }
    }
}
impl std::error::Error for UnitError {}
