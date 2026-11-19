// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    history::SketchConstraintReservationError,
    identity::RecipeIdError,
    parameter::ParameterError,
    sketch::{SketchError, SketchTransientSolveError},
};
use std::{error::Error, fmt};
/// Source, geometry and permanent allocation failures keep their distinct causes.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchDimensionCreationError {
    ChoiceRequired,
    Sketch(SketchError),
    Inference(SketchTransientSolveError),
    Readiness(crate::document_sketch::SketchReadinessError),
    Parameter(ParameterError<u64>),
    Identity(RecipeIdError),
    ConstraintIdentity(SketchConstraintReservationError),
}
impl SketchDimensionCreationError {
    fn cause(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::ChoiceRequired => None,
            Self::Sketch(cause) => Some(cause),
            Self::Inference(cause) => Some(cause),
            Self::Readiness(cause) => Some(cause),
            Self::Parameter(cause) => Some(cause),
            Self::Identity(cause) => Some(cause),
            Self::ConstraintIdentity(cause) => Some(cause),
        }
    }
}
impl fmt::Display for SketchDimensionCreationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self.cause() {
            Some(cause) => write!(output, "dimension creation: {cause}"),
            None => output.write_str("dimension creation would over-constrain the sketch: expected a driven dimension choice"),
        }
    }
}
impl Error for SketchDimensionCreationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        self.cause()
    }
}
impl From<SketchError> for SketchDimensionCreationError {
    fn from(cause: SketchError) -> Self {
        Self::Sketch(cause)
    }
}
impl From<ParameterError<u64>> for SketchDimensionCreationError {
    fn from(cause: ParameterError<u64>) -> Self {
        Self::Parameter(cause)
    }
}
impl From<RecipeIdError> for SketchDimensionCreationError {
    fn from(cause: RecipeIdError) -> Self {
        Self::Identity(cause)
    }
}
impl From<SketchConstraintReservationError> for SketchDimensionCreationError {
    fn from(cause: SketchConstraintReservationError) -> Self {
        Self::ConstraintIdentity(cause)
    }
}
