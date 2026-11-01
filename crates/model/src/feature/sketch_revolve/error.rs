// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    document_sketch::DocumentSketchId, feature::work_geometry::WorkGeometryError,
    history::DocumentSketchPlacementError, sketch::SketchError,
};
use std::{error::Error, fmt};

/// A refusal retains the sketch and the specific source or current-resolution error.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchRevolveError {
    pub sketch: DocumentSketchId,
    pub cause: Box<SketchRevolveCause>,
}

/// Typed failure of authored revolve inputs or their current evaluation.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchRevolveCause {
    Definition(String),
    Placement(DocumentSketchPlacementError),
    Sketch(SketchError),
    Angle(WorkGeometryError),
}

impl fmt::Display for SketchRevolveError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            output,
            "document sketch {:?} revolve: {}",
            self.sketch, self.cause
        )
    }
}

impl fmt::Display for SketchRevolveCause {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Definition(message) => output.write_str(message),
            Self::Placement(cause) => cause.fmt(output),
            Self::Sketch(cause) => cause.fmt(output),
            Self::Angle(cause) => cause.fmt(output),
        }
    }
}

impl Error for SketchRevolveError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(self.cause.as_ref())
    }
}

impl Error for SketchRevolveCause {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Definition(_) => None,
            Self::Placement(cause) => Some(cause),
            Self::Sketch(cause) => Some(cause),
            Self::Angle(cause) => Some(cause),
        }
    }
}
