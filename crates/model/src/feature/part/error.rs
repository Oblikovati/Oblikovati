// SPDX-License-Identifier: GPL-2.0-only
use crate::feature::{
    FeatureId, FeatureReplayError, FeatureSequenceError, sketch_prism::SketchPrismError,
};
use std::{error::Error, fmt};
#[derive(Clone, Debug, PartialEq)]
pub enum PartFeatureProgramError {
    Prism {
        feature: FeatureId,
        cause: SketchPrismError,
    },
    Sequence(FeatureSequenceError),
    Reads(FeatureReplayError),
}
impl fmt::Display for PartFeatureProgramError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Prism { feature, cause } => write!(output, "part feature {feature:?}: {cause}"),
            Self::Sequence(cause) => cause.fmt(output),
            Self::Reads(cause) => cause.fmt(output),
        }
    }
}
impl Error for PartFeatureProgramError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Prism { cause, .. } => Some(cause),
            Self::Sequence(cause) => Some(cause),
            Self::Reads(cause) => Some(cause),
        }
    }
}
