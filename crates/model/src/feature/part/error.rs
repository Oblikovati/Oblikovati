// SPDX-License-Identifier: GPL-2.0-only
use crate::feature::{
    FeatureId, FeatureReplayError, FeatureSequenceError, sketch_prism::SketchPrismError,
};
use std::{error::Error, fmt};
#[derive(Clone, Debug, PartialEq)]
pub enum PartFeatureProgramError {
    Cancelled {
        completed: usize,
    },
    Prism {
        feature: FeatureId,
        cause: SketchPrismError,
    },
    Sweep {
        feature: FeatureId,
        cause: Box<crate::feature::sketch_sweep::SketchSweepError>,
    },
    Loft {
        feature: FeatureId,
        cause: String,
    },
    Revolve {
        feature: FeatureId,
        cause: Box<crate::feature::sketch_revolve::SketchRevolveError>,
    },
    Coil {
        feature: FeatureId,
        cause: String,
    },
    Emboss {
        feature: FeatureId,
        cause: String,
    },
    Rib {
        feature: FeatureId,
        cause: String,
    },
    Decal {
        feature: FeatureId,
        cause: String,
    },
    Mark {
        feature: FeatureId,
        cause: String,
    },
    Sequence(FeatureSequenceError),
    Reads(FeatureReplayError),
}
impl fmt::Display for PartFeatureProgramError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Cancelled { completed } => write!(
                output,
                "part replay cancelled after {completed} entries: expected a complete current generation"
            ),
            Self::Prism { feature, cause } => write!(output, "part feature {feature:?}: {cause}"),
            Self::Sweep { feature, cause } => write!(output, "part feature {feature:?}: {cause}"),
            Self::Loft { feature, cause } => write!(output, "part Loft {feature:?}: {cause}"),
            Self::Revolve { feature, cause } => {
                write!(output, "part Revolve {feature:?}: {cause}")
            }
            Self::Coil { feature, cause } => write!(output, "part Coil {feature:?}: {cause}"),
            Self::Emboss { feature, cause } => write!(output, "part Emboss {feature:?}: {cause}"),
            Self::Mark { feature, cause } => write!(output, "part Mark {feature:?}: {cause}"),
            Self::Rib { feature, cause } => write!(output, "part Rib {feature:?}: {cause}"),
            Self::Decal { feature, cause } => write!(output, "part Decal {feature:?}: {cause}"),
            Self::Sequence(cause) => cause.fmt(output),
            Self::Reads(cause) => cause.fmt(output),
        }
    }
}
impl Error for PartFeatureProgramError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Cancelled { .. }
            | Self::Loft { .. }
            | Self::Coil { .. }
            | Self::Emboss { .. }
            | Self::Rib { .. }
            | Self::Decal { .. }
            | Self::Mark { .. } => None,
            Self::Prism { cause, .. } => Some(cause),
            Self::Sweep { cause, .. } => Some(cause.as_ref()),
            Self::Revolve { cause, .. } => Some(cause.as_ref()),
            Self::Sequence(cause) => Some(cause),
            Self::Reads(cause) => Some(cause),
        }
    }
}
