// SPDX-License-Identifier: GPL-2.0-only
//! Resolve feature suppression in the active part member.
use super::DocumentSnapshot;
use crate::feature::{FeatureId, FeatureSequenceError};

impl DocumentSnapshot {
    /// Build the current member's replay recipe. Example: `snapshot.effective_work_geometry()`.
    pub fn effective_work_geometry(
        &self,
    ) -> crate::feature::FeatureSequence<crate::feature::work_geometry::WorkGeometryDefinition>
    {
        self.work_geometry
            .with_effective_suppression(|id, authored| {
                self.model_states
                    .as_ref()
                    .map_or(authored, |states| states.suppressed(id.0, authored))
            })
    }
    /// Read effective suppression in the active member. Example: `snapshot.feature_suppressed(id)?`.
    pub fn feature_suppressed(&self, id: FeatureId) -> Result<bool, FeatureSequenceError> {
        let authored = self.work_geometry.get(id)?.suppressed();
        Ok(self
            .model_states
            .as_ref()
            .map_or(authored, |states| states.suppressed(id.0, authored)))
    }
    /// Edit suppression in the current member scope. Example: `snapshot.set_feature_suppressed(id, true)?`.
    pub fn set_feature_suppressed(
        &mut self,
        id: FeatureId,
        suppressed: bool,
    ) -> Result<(), FeatureSequenceError> {
        let authored = self.work_geometry.get(id)?.suppressed();
        if let Some(states) = &mut self.model_states {
            states.set_suppressed(id.0, authored, suppressed);
        } else {
            self.work_geometry.set_suppressed(id, suppressed)?;
        }
        Ok(())
    }
}
