// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    part::{PartFeatureDefinition, PartFeatureProgramError},
    sketch_sweep::SketchSweepDefinition,
};

impl DocumentSnapshot {
    /// Append the two Sweep sources to a private transaction candidate.
    /// Example: `snapshot.create_sketch_sweep("Sweep1", definition, ids)?`.
    pub fn create_sketch_sweep(
        &mut self,
        name: &str,
        definition: SketchSweepDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let reads = definition.reads(self).map_err(|cause| {
            HistoryError::PartProgram(PartFeatureProgramError::Sweep {
                feature: id,
                cause: Box::new(cause),
            })
        })?;
        self.work_geometry
            .append(AuthoredFeature::new(
                id,
                name,
                PartFeatureDefinition::SketchSweep(Box::new(definition)),
                reads,
            ))
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))?;
        if let Some(states) = &mut self.model_states {
            states.register_feature(id.0);
        }
        Ok(id)
    }
}
