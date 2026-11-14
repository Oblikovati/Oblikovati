// SPDX-License-Identifier: GPL-2.0-only
use super::DatumLength;
use crate::feature::work_geometry::DatumLengthError;
use crate::{
    document_sketch::DocumentSketchId, history::DocumentSketchPlacementError, sketch::SketchError,
};
use std::{error::Error, fmt};
/// A refusal retains the source sketch and typed placement or scalar failure.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPrismError {
    pub sketch: DocumentSketchId,
    // Keep the successful return path small without discarding typed failure evidence.
    pub cause: Box<SketchPrismCause>,
}
#[derive(Clone, Debug, PartialEq)]
pub enum SketchPrismCause {
    Definition(String),
    Placement(DocumentSketchPlacementError),
    Sketch(SketchError),
    Length {
        axis: usize,
        source: DatumLength,
        cause: DatumLengthError,
    },
}
impl fmt::Display for SketchPrismError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            output,
            "document sketch {:?} prism: {}",
            self.sketch, self.cause
        )
    }
}
impl fmt::Display for SketchPrismCause {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Definition(message) => write!(output, "{message}"),
            Self::Placement(cause) => cause.fmt(output),
            Self::Sketch(cause) => cause.fmt(output),
            Self::Length {
                axis,
                source,
                cause,
            } => write!(
                output,
                "local displacement axis {axis}, source {source:?}: {cause}"
            ),
        }
    }
}
impl Error for SketchPrismError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(self.cause.as_ref())
    }
}
impl Error for SketchPrismCause {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Definition(_) => None,
            Self::Placement(cause) => Some(cause),
            Self::Sketch(cause) => Some(cause),
            Self::Length { cause, .. } => Some(cause),
        }
    }
}
