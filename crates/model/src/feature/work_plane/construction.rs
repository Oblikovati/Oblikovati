// SPDX-License-Identifier: GPL-2.0-only
use super::WorkPlaneDefinition;
use crate::feature::{AuthoredFeature, FeatureId, FeatureSequence, FeatureSequenceError};

impl AuthoredFeature<WorkPlaneDefinition> {
    /// Read consumer-tied lifecycle state separately from the frame; example: `plane.is_construction_datum()`.
    pub fn is_construction_datum(&self) -> bool {
        self.construction_datum
    }
}
impl FeatureSequence<WorkPlaneDefinition> {
    /// Set authored lifecycle state without changing geometry or dependencies; example: `planes.set_work_plane_construction(id, true)?`.
    pub fn set_work_plane_construction(
        &mut self,
        id: FeatureId,
        construction: bool,
    ) -> Result<(), FeatureSequenceError> {
        let position = self.position(id)?;
        self.features[position].construction_datum = construction;
        Ok(())
    }
}
