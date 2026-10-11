// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    history::SketchConstraintReservationError, identity::RecipeIdError, parameter::ParameterError,
    sketch::SketchError,
};
use std::{error::Error, fmt};
/// Source, geometry and permanent allocation failures keep their distinct causes.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchDimensionCreationError {
    Sketch(SketchError),
    Parameter(ParameterError<u64>),
    Identity(RecipeIdError),
    ConstraintIdentity(SketchConstraintReservationError),
}
impl SketchDimensionCreationError {
    fn cause(&self) -> &(dyn Error + 'static) {
        match self {
            Self::Sketch(cause) => cause,
            Self::Parameter(cause) => cause,
            Self::Identity(cause) => cause,
            Self::ConstraintIdentity(cause) => cause,
        }
    }
}
impl fmt::Display for SketchDimensionCreationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(output, "dimension creation: {}", self.cause())
    }
}
impl Error for SketchDimensionCreationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(self.cause())
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
