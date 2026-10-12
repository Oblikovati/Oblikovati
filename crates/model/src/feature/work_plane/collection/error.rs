// SPDX-License-Identifier: GPL-2.0-only

use crate::feature::{
    FeatureId, FeatureReplayError, FeatureSequenceError, FeatureSuppressionError,
    work_plane::WorkPlaneError,
};
use std::fmt::{Display, Formatter};

/// A rejected collection cannot replace live feature state.
#[derive(Clone, Debug, PartialEq)]
pub enum WorkPlaneProgramRecipeError {
    Version(u32),
    LegacyConstruction(FeatureId),
    LegacyVisibility(FeatureId),
    LegacyDefinition { feature: FeatureId, minimum: u32 },
    Plane(WorkPlaneError),
    Condition(FeatureSuppressionError),
    Sequence(FeatureSequenceError),
    Reads(FeatureReplayError),
}

impl Display for WorkPlaneProgramRecipeError {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::LegacyDefinition { feature, minimum } => write!(
                formatter,
                "datum feature {feature:?}: expected program version {minimum}"
            ),
            Self::Version(version) => unsupported_version(*version, formatter),
            Self::LegacyVisibility(id) => write_visibility(*id, formatter),
            Self::LegacyConstruction(id) => write!(
                formatter,
                "work plane {} construction=true in program version 1: expected version 2",
                id.0
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
            Self::Version(_)
            | Self::LegacyConstruction(_)
            | Self::LegacyVisibility(_)
            | Self::LegacyDefinition { .. } => None,
            Self::Plane(error) => Some(error),
            Self::Condition(error) => Some(error),
            Self::Sequence(error) => Some(error),
            Self::Reads(error) => Some(error),
        }
    }
}

fn unsupported_version(version: u32, formatter: &mut Formatter<'_>) -> std::fmt::Result {
    write!(
        formatter,
        "work plane program version {version}: expected 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, or 16"
    )
}

fn write_visibility(id: FeatureId, formatter: &mut Formatter<'_>) -> std::fmt::Result {
    write!(
        formatter,
        "datum {id:?} visible=false: expected program version 5"
    )
}
