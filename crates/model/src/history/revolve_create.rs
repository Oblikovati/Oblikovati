// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    part::{PartFeatureDefinition, PartFeatureProgramError},
    sketch_revolve::SketchRevolveDefinition,
};

impl DocumentSnapshot {
    /// Append a Revolve to a private candidate. Example: `snapshot.create_sketch_revolve("Revolution1", definition, ids)?`.
    pub fn create_sketch_revolve(
        &mut self,
        name: &str,
        definition: SketchRevolveDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let reads = definition.reads(self).map_err(|cause| {
            HistoryError::PartProgram(PartFeatureProgramError::Revolve {
                feature: id,
                cause: Box::new(cause),
            })
        })?;
        let source = definition.sketch();
        self.work_geometry
            .append(AuthoredFeature::new(
                id,
                name,
                PartFeatureDefinition::SketchRevolve(Box::new(definition)),
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
