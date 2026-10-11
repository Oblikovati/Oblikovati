// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    document_sketch::DocumentSketchId,
    feature::{FeatureReadKey, FeatureSequenceError},
    sketch::SketchError,
};
use std::{error::Error, fmt};

/// A deletion decline names the sketch, dependent, or inconsistent incidence before owner adoption.
#[derive(Clone, Debug, PartialEq)]
pub enum DocumentSketchDeleteError {
    Sketch(SketchError),
    Datum(FeatureSequenceError),
    Consumed {
        sketch: DocumentSketchId,
        readers: Vec<FeatureReadKey>,
    },
    Unattributed {
        sketch: DocumentSketchId,
        readers: Vec<FeatureReadKey>,
    },
    MissingIncidence(FeatureReadKey),
}
impl fmt::Display for DocumentSketchDeleteError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Sketch(cause) => cause.fmt(output),
            Self::Datum(cause) => cause.fmt(output),
            Self::Consumed { sketch, readers } => {
                dependent_decline(output, *sketch, readers, "consumed")
            }
            Self::Unattributed { sketch, readers } => {
                dependent_decline(output, *sketch, readers, "unattributed")
            }
            Self::MissingIncidence(key) => write!(
                output,
                "consumer incidence {key:?}: expected a stored source and matching target edges"
            ),
        }
    }
}
impl Error for DocumentSketchDeleteError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Sketch(cause) => Some(cause),
            Self::Datum(cause) => Some(cause),
            _ => None,
        }
    }
}
fn dependent_decline(
    output: &mut fmt::Formatter<'_>,
    sketch: DocumentSketchId,
    readers: &[FeatureReadKey],
    state: &str,
) -> fmt::Result {
    write!(
        output,
        "sketch {} deletion is {state} by {readers:?}: expected no consumers and complete read attribution",
        sketch.0
    )
}
