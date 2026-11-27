// SPDX-License-Identifier: GPL-2.0-only
//! Append a Derived Part feature to a Part candidate snapshot.
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    derived_part::DerivedPartDefinition,
    part::{PartFeatureDefinition, PartFeatureProgramError},
};

impl DocumentSnapshot {
    /// Append a Derived Part feature to a private candidate.
    /// Example: `snapshot.create_derived_part("Part7.opd", definition, ids)?`.
    pub fn create_derived_part(
        &mut self,
        name: &str,
        definition: DerivedPartDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let reads = definition.source_reads();
        self.work_geometry
            .append(AuthoredFeature::new(
                id,
                name,
                PartFeatureDefinition::DerivedPart(Box::new(definition)),
                reads,
            ))
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))?;
        if let Some(states) = &mut self.model_states {
            states.register_feature(id.0);
        }
        Ok(id)
    }
}
