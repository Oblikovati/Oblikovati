// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    feature::{FeatureReadKey, FeatureScalarOwnershipError, FeatureSequenceError},
    parameter::ParameterError,
};
use std::{error::Error, fmt};

/// An unused-datum proof failed; the original document remains unchanged.
#[derive(Clone, Debug, PartialEq)]
pub enum DocumentDatumDeleteError {
    Ownership(FeatureScalarOwnershipError),
    Feature(FeatureSequenceError),
    Parameter(ParameterError<u64>),
    Consumed {
        target: FeatureReadKey,
        readers: Vec<FeatureReadKey>,
    },
}
impl fmt::Display for DocumentDatumDeleteError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Ownership(error) => error.fmt(output),
            Self::Feature(error) => error.fmt(output),
            Self::Parameter(error) => error.fmt(output),
            Self::Consumed { target, readers } => write!(
                output,
                "datum deletion target {target:?} has consumers {readers:?}: expected no external consumers and complete read attribution"
            ),
        }
    }
}
impl Error for DocumentDatumDeleteError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Ownership(error) => Some(error),
            Self::Feature(error) => Some(error),
            Self::Parameter(error) => Some(error),
            Self::Consumed { .. } => None,
        }
    }
}
