// SPDX-License-Identifier: GPL-2.0-only
use super::SketchConstraintId;
use crate::parameter::ParameterError;
use oblikovati_solve::nonlinear::SolveError;
use std::{error::Error, fmt};

/// Invalid authored references or quantities, or a failed numerical evaluation.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchError {
    Definition(String),
    Parameter {
        constraint: SketchConstraintId,
        parameter: u64,
        cause: ParameterError<u64>,
    },
    Numerical(SolveError),
}
impl fmt::Display for SketchError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Definition(message) => message.fmt(output),
            Self::Parameter {
                constraint,
                parameter,
                cause,
            } => write!(
                output,
                "sketch constraint {constraint:?} length parameter {parameter}: {cause}"
            ),
            Self::Numerical(cause) => cause.fmt(output),
        }
    }
}
impl Error for SketchError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Definition(_) => None,
            Self::Parameter { cause, .. } => Some(cause),
            Self::Numerical(cause) => Some(cause),
        }
    }
}
impl From<SolveError> for SketchError {
    fn from(cause: SolveError) -> Self {
        Self::Numerical(cause)
    }
}
