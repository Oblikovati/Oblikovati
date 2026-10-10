// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchConstraintId, SketchEntityId};
use crate::parameter::ParameterError;
use oblikovati_solve::{
    nonlinear::SolveError,
    residual::{EvaluationError, ResidualError},
};
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
    ContactSelection {
        constraint: SketchConstraintId,
        cause: ResidualError,
    },
    ArcCircularity {
        arc: SketchEntityId,
        cause: EvaluationError,
    },
    Residual {
        constraint: SketchConstraintId,
        cause: EvaluationError,
    },
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
            Self::ContactSelection { constraint, cause } => write!(
                output,
                "sketch constraint {constraint:?} contact selection: {cause}"
            ),
            Self::ArcCircularity { arc, cause } => {
                write!(output, "sketch arc {arc:?} circularity: {cause}")
            }
            Self::Residual { constraint, cause } => {
                write!(output, "sketch constraint {constraint:?}: {cause}")
            }
        }
    }
}
impl Error for SketchError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Definition(_) => None,
            Self::Parameter { cause, .. } => Some(cause),
            Self::Numerical(cause) => Some(cause),
            Self::ContactSelection { cause, .. } => Some(cause),
            Self::Residual { cause, .. } | Self::ArcCircularity { cause, .. } => Some(cause),
        }
    }
}
impl From<SolveError> for SketchError {
    fn from(cause: SolveError) -> Self {
        Self::Numerical(cause)
    }
}
