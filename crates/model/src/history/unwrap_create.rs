// SPDX-License-Identifier: GPL-2.0-only
//! Append an Unwrap feature to a Part candidate snapshot.
use super::super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    part::{PartFeatureDefinition, PartFeatureProgramError},
    unwrap::UnwrapDefinition,
};

impl DocumentSnapshot {
    /// Append an Unwrap feature to a private candidate.
    /// Example: `snapshot.create_unwrap("Unwrap1", definition, ids)?`.
    pub fn create_unwrap(
        &mut self,
        name: &str,
        definition: UnwrapDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let reads = definition.source_reads();
        self.work_geometry
            .append(AuthoredFeature::new(
                id,
                name,
                PartFeatureDefinition::Unwrap(Box::new(definition)),
                reads,
            ))
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))?;
        if let Some(states) = &mut self.model_states {
            states.register_feature(id.0);
        }
        Ok(id)
    }
}
