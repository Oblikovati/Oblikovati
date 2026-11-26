// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    part::{PartFeatureDefinition, PartFeatureProgramError},
    sketch_loft::SketchLoftDefinition,
};

impl DocumentSnapshot {
    /// Append a Loft to a private candidate. Example: `snapshot.create_sketch_loft("Loft1", definition, ids)?`.
    pub fn create_sketch_loft(
        &mut self,
        name: &str,
        definition: SketchLoftDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let reads = definition.reads(self).map_err(|cause| {
            HistoryError::PartProgram(PartFeatureProgramError::Loft { feature: id, cause })
        })?;
        let sources = definition.recipe().sections;
        self.work_geometry
            .append(AuthoredFeature::new(
                id,
                name,
                PartFeatureDefinition::SketchLoft(Box::new(definition)),
                reads,
            ))
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))?;
        if let Some(states) = &mut self.model_states {
            states.register_feature(id.0);
        }
        for section in sources {
            self.sketches
                .set_visible(section.sketch, false)
                .map_err(|_| HistoryError::UnreservedSketchId(section.sketch.0))?;
        }
        Ok(id)
    }
}
