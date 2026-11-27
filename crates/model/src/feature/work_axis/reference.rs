// SPDX-License-Identifier: GPL-2.0-only
use super::WorkAxisLine;
use crate::feature::{
    FeatureId,
    work_plane::{WorkPlaneError, WorkPlaneResolver},
};

/// One stable axis reference shared by datums and sketch orientation.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum WorkAxisReference {
    OriginX,
    OriginY,
    OriginZ,
    Feature(FeatureId),
}
impl WorkAxisReference {
    /// Resolve the complete current line; example: `WorkAxisReference::OriginX.line(&evaluation)?`.
    pub fn line(self, resolver: &impl WorkPlaneResolver) -> Result<WorkAxisLine, WorkPlaneError> {
        let direction = match self {
            Self::OriginX => [1.0, 0.0, 0.0],
            Self::OriginY => [0.0, 1.0, 0.0],
            Self::OriginZ => [0.0, 0.0, 1.0],
            Self::Feature(id) => return resolver.axis(id),
        };
        WorkAxisLine::new([0.0; 3], direction).map_err(WorkPlaneError::Axis)
    }
}
