// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    history::{
        SketchConstraintReservationError, SketchDimensionCreationError,
        SketchPrimitiveCreationError,
    },
    sketch::SketchError,
};
use std::{error::Error, fmt};
/// Typed failures retain the failing geometry, reservation, or field dimension cause.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchPlacementCreationError {
    Sketch(SketchError),
    Primitive(SketchPrimitiveCreationError),
    Constraint(SketchConstraintReservationError),
    Field {
        label: String,
        cause: SketchDimensionCreationError,
    },
}
impl SketchPlacementCreationError {
    fn cause(&self) -> &(dyn Error + 'static) {
        match self {
            Self::Sketch(cause) => cause,
            Self::Primitive(cause) => cause,
            Self::Constraint(cause) => cause,
            Self::Field { cause, .. } => cause,
        }
    }
}
impl fmt::Display for SketchPlacementCreationError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if let Self::Field { label, cause } = self {
            return write!(f, "placement field {label:?}: {cause}");
        }
        write!(f, "sketch placement creation: {}", self.cause())
    }
}
impl Error for SketchPlacementCreationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(self.cause())
    }
}
impl From<SketchError> for SketchPlacementCreationError {
    fn from(cause: SketchError) -> Self {
        Self::Sketch(cause)
    }
}
