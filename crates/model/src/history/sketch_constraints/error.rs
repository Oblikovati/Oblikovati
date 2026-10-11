// SPDX-License-Identifier: GPL-2.0-only
use crate::{document_sketch::DocumentSketchId, identity::RecipeIdError};
use std::{error::Error, fmt};
/// A local allocation failure retains the sketch namespace for command diagnostics.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchConstraintReservationError {
    pub sketch: DocumentSketchId,
    pub cause: RecipeIdError,
}
impl fmt::Display for SketchConstraintReservationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            output,
            "sketch {} constraint reservation: expected a nonzero reserved sketch namespace with an available constraint ID: {}",
            self.sketch.0, self.cause
        )
    }
}
impl Error for SketchConstraintReservationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(&self.cause)
    }
}
