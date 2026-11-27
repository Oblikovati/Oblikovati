// SPDX-License-Identifier: GPL-2.0-only
//! Append an Imported Model feature to a Part candidate snapshot.
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    imported_model::ImportedModelDefinition,
    part::{PartFeatureDefinition, PartFeatureProgramError},
};

impl DocumentSnapshot {
    /// Append an Imported Model feature to a private candidate.
    /// Example: `snapshot.create_imported_model("Solid1:1", definition, ids)?`.
    pub fn create_imported_model(
        &mut self,
        name: &str,
        definition: ImportedModelDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let reads = definition.source_reads();
        self.work_geometry
            .append(AuthoredFeature::new(
                id,
                name,
                PartFeatureDefinition::ImportedModel(Box::new(definition)),
                reads,
            ))
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))?;
        if let Some(states) = &mut self.model_states {
            states.register_feature(id.0);
        }
        Ok(id)
    }
}
