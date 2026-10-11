// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    history::{SketchConstraintReservationError, SketchPrimitiveCreationError},
    sketch::SketchError,
};
use std::{error::Error, fmt};
/// Keep geometry, primitive reservations, and local constraint exhaustion as typed causes.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchLineCreationError {
    Sketch(SketchError),
    Primitive(SketchPrimitiveCreationError),
    Constraint(SketchConstraintReservationError),
}
impl SketchLineCreationError {
    fn cause(&self) -> &(dyn Error + 'static) {
        match self {
            Self::Sketch(cause) => cause,
            Self::Primitive(cause) => cause,
            Self::Constraint(cause) => cause,
        }
    }
}
impl fmt::Display for SketchLineCreationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(output, "inferred line creation: {}", self.cause())
    }
}
impl Error for SketchLineCreationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(self.cause())
    }
}
