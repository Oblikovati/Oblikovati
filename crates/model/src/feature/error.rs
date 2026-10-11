// SPDX-License-Identifier: GPL-2.0-only

use super::FeatureId;
use crate::dependency::DependencyError;
use std::fmt::{Display, Formatter};

/// Invalid authored edits leave the sequence and its dependency graph unchanged.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum FeatureSequenceError {
    Dependency(DependencyError<FeatureId>),
    Index {
        index: usize,
        length: usize,
    },
    Boundary {
        index: usize,
        length: usize,
    },
    ForwardRead {
        feature: FeatureId,
        driver: FeatureId,
    },
}

impl Display for FeatureSequenceError {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Dependency(error) => Display::fmt(error, formatter),
            Self::Index { index, length } => {
                write!(formatter, "feature index {index}: expected [0,{length})")
            }
            Self::Boundary { index, length } => write!(
                formatter,
                "end-of-part index {index}: expected [0,{length}]"
            ),
            Self::ForwardRead { feature, driver } => write!(
                formatter,
                "feature {feature:?} reads {driver:?}: expected driver before reader"
            ),
        }
    }
}

impl std::error::Error for FeatureSequenceError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Dependency(error) => Some(error),
            _ => None,
        }
    }
}

impl From<DependencyError<FeatureId>> for FeatureSequenceError {
    fn from(error: DependencyError<FeatureId>) -> Self {
        Self::Dependency(error)
    }
}
