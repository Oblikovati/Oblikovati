// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    history::{SketchConstraintReservationError, SketchPrimitiveCreationError},
    sketch::SketchError,
};
use std::{error::Error, fmt};
/// Retain distinct geometry, global entity, and local constraint reservation failures.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchPolygonCreationError {
    Sketch(SketchError),
    Primitive(SketchPrimitiveCreationError),
    Constraint(SketchConstraintReservationError),
}
impl SketchPolygonCreationError {
    fn cause(&self) -> &(dyn Error + 'static) {
        match self {
            Self::Sketch(cause) => cause,
            Self::Primitive(cause) => cause,
            Self::Constraint(cause) => cause,
        }
    }
}
impl fmt::Display for SketchPolygonCreationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(output, "regular polygon creation: {}", self.cause())
    }
}
impl Error for SketchPolygonCreationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(self.cause())
    }
}
