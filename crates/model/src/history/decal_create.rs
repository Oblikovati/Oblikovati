// SPDX-License-Identifier: GPL-2.0-only
//! Append a Decal feature to a Part candidate snapshot.
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    part::{PartFeatureDefinition, PartFeatureProgramError},
    sketch_decal::SketchDecalDefinition,
};

impl DocumentSnapshot {
    /// Append a Decal feature to a private candidate.
    /// Example: `snapshot.create_sketch_decal("Decal1", definition, ids)?`.
    pub fn create_sketch_decal(
        &mut self,
        name: &str,
        definition: SketchDecalDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let reads = definition.reads(self).map_err(|cause| {
            HistoryError::PartProgram(PartFeatureProgramError::Decal { feature: id, cause })
        })?;
        let source = definition.sketch();
        self.work_geometry
            .append(AuthoredFeature::new(
                id,
                name,
                PartFeatureDefinition::SketchDecal(Box::new(definition)),
                reads,
            ))
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))?;
        if let Some(states) = &mut self.model_states {
            states.register_feature(id.0);
        }
        self.sketches
            .set_visible(source, false)
            .map_err(|_| HistoryError::UnreservedSketchId(source.0))?;
        Ok(id)
    }
}
