// SPDX-License-Identifier: GPL-2.0-only
use crate::{parameter::ParameterError, quantity::Quantity};
use std::{error::Error, fmt};

/// A shared scalar decline retains the selected parameter and offending quantity.
#[derive(Clone, Debug, PartialEq)]
pub enum DatumLengthError {
    Parameter(ParameterError<u64>),
    Quantity {
        parameter: Option<u64>,
        quantity: Quantity,
    },
}
impl fmt::Display for DatumLengthError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Parameter(cause) => cause.fmt(output),
            Self::Quantity {
                parameter,
                quantity,
            } => write!(
                output,
                "datum length parameter {parameter:?} value {quantity:?}: expected finite Length in document working units"
            ),
        }
    }
}
impl Error for DatumLengthError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Parameter(cause) => Some(cause),
            _ => None,
        }
    }
}
