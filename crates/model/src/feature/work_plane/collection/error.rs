// SPDX-License-Identifier: GPL-2.0-only

use crate::feature::{
    FeatureReplayError, FeatureSequenceError, FeatureSuppressionError, work_plane::WorkPlaneError,
};
use std::fmt::{Display, Formatter};

/// A rejected collection cannot replace live feature state.
#[derive(Clone, Debug, PartialEq)]
pub enum WorkPlaneProgramRecipeError {
    Version(u32),
    Plane(WorkPlaneError),
    Condition(FeatureSuppressionError),
    Sequence(FeatureSequenceError),
    Reads(FeatureReplayError),
}

impl Display for WorkPlaneProgramRecipeError {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Version(version) => write!(
                formatter,
                "work plane program version {version}: expected 1"
            ),
            Self::Plane(error) => Display::fmt(error, formatter),
            Self::Condition(error) => Display::fmt(error, formatter),
            Self::Sequence(error) => Display::fmt(error, formatter),
            Self::Reads(error) => Display::fmt(error, formatter),
        }
    }
}

impl std::error::Error for WorkPlaneProgramRecipeError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Version(_) => None,
            Self::Plane(error) => Some(error),
            Self::Condition(error) => Some(error),
            Self::Sequence(error) => Some(error),
            Self::Reads(error) => Some(error),
        }
    }
}
