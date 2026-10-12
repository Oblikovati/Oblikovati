// SPDX-License-Identifier: GPL-2.0-only
use super::{AuthoredFeature, FeatureSequence, WorkGeometryDefinition, WorkGeometryError};
impl AuthoredFeature<WorkGeometryDefinition> {
    pub(crate) fn validate_plane_metadata(&self) -> Result<(), WorkGeometryError> {
        if self.construction_datum
            && matches!(self.definition(), WorkGeometryDefinition::SketchPrism(_))
        {
            return Err(WorkGeometryError::Kind(super::WorkGeometryKindMismatch {
                feature: self.id(),
                expected: "construction datum",
                actual: "sketch prism input",
            }));
        }
        self.validate_plane_orientation()?;
        self.validate_plane_display()
    }
}
impl FeatureSequence<WorkGeometryDefinition> {
    /// Reject plane-only metadata on other datum kinds; example: `datums.validate_plane_metadata()?` before adoption.
    pub fn validate_plane_metadata(&self) -> Result<(), WorkGeometryError> {
        for feature in self.features() {
            feature.validate_plane_metadata()?;
        }
        Ok(())
    }
}
