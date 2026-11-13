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
    ProjectionSource {
        point: SketchEntityId,
        cause: Box<crate::history::DocumentSketchPlacementError>,
    },
    PatternParameter {
        member: SketchEntityId,
        parameter: u64,
        cause: Box<ParameterError<u64>>,
    },
    Parameter {
        constraint: SketchConstraintId,
        parameter: u64,
        cause: ParameterError<u64>,
    },
    Numerical(SolveError),
    AngleSelection {
        constraint: SketchConstraintId,
        cause: ResidualError,
    },
    ContactSelection {
        constraint: SketchConstraintId,
        cause: ResidualError,
    },
    ArcMidpoint {
        arc: SketchEntityId,
        cause: ResidualError,
    },
    PatternRadiusLink {
        member: SketchEntityId,
        cause: EvaluationError,
    },
    PatternPointLink {
        member: SketchEntityId,
        cause: EvaluationError,
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
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Definition(message) => message.fmt(f),
            Self::ProjectionSource { point, cause } => point_failure(f, point, cause),
            Self::PatternParameter {
                member,
                parameter,
                cause,
            } => write!(f, "pattern point {member:?} parameter {parameter}: {cause}"),
            Self::Parameter {
                constraint,
                parameter,
                cause,
            } => write_parameter_failure(f, *constraint, *parameter, cause),
            Self::Numerical(cause) => cause.fmt(f),
            Self::AngleSelection { constraint, cause }
            | Self::ContactSelection { constraint, cause } => {
                relation_error(f, constraint, self.selection_label(), cause)
            }
            Self::ArcMidpoint { arc, cause } => arc_error(f, arc, "midpoint", cause),
            Self::ArcCircularity { arc, cause } => arc_error(f, arc, "circularity", cause),
            Self::PatternRadiusLink { member, cause } => {
                write!(f, "pattern circle {member:?} radius link: {cause}")
            }
            Self::PatternPointLink { member, cause } => {
                write!(f, "pattern point {member:?} link: {cause}")
            }
            Self::Residual { constraint, cause } => relation_error(f, constraint, None, cause),
        }
    }
}
impl SketchError {
    fn selection_label(&self) -> Option<&'static str> {
        match self {
            Self::AngleSelection { .. } => Some("angle"),
            Self::ContactSelection { .. } => Some("contact"),
            _ => None,
        }
    }
}
fn point_failure(
    f: &mut fmt::Formatter<'_>,
    point: &SketchEntityId,
    cause: &crate::history::DocumentSketchPlacementError,
) -> fmt::Result {
    write!(f, "sketch point {point:?} projection source: {cause}")
}
fn write_parameter_failure(
    f: &mut fmt::Formatter<'_>,
    constraint: SketchConstraintId,
    parameter: u64,
    cause: &ParameterError<u64>,
) -> fmt::Result {
    write!(
        f,
        "sketch constraint {constraint:?} dimension parameter {parameter}: {cause}"
    )
}
fn relation_error<Cause: fmt::Display>(
    f: &mut fmt::Formatter<'_>,
    constraint: &SketchConstraintId,
    selection: Option<&str>,
    cause: &Cause,
) -> fmt::Result {
    if let Some(selection) = selection {
        return write!(
            f,
            "sketch constraint {constraint:?} {selection} selection: {cause}"
        );
    }
    write!(f, "sketch constraint {constraint:?}: {cause}")
}
fn arc_error<Cause: fmt::Display>(
    f: &mut fmt::Formatter<'_>,
    arc: &SketchEntityId,
    operation: &str,
    cause: &Cause,
) -> fmt::Result {
    write!(f, "sketch arc {arc:?} {operation}: {cause}")
}
impl Error for SketchError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Definition(_) => None,
            Self::ProjectionSource { cause, .. } => Some(cause.as_ref()),
            Self::Parameter { cause, .. } => Some(cause),
            Self::PatternParameter { cause, .. } => Some(cause.as_ref()),
            Self::Numerical(cause) => Some(cause),
            Self::AngleSelection { cause, .. }
            | Self::ContactSelection { cause, .. }
            | Self::ArcMidpoint { cause, .. } => Some(cause),
            Self::Residual { cause, .. }
            | Self::ArcCircularity { cause, .. }
            | Self::PatternPointLink { cause, .. }
            | Self::PatternRadiusLink { cause, .. } => Some(cause),
        }
    }
}
impl From<SolveError> for SketchError {
    fn from(cause: SolveError) -> Self {
        Self::Numerical(cause)
    }
}
