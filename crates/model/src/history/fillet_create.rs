// SPDX-License-Identifier: GPL-2.0-only
//! Create authored fillet inputs in the common Part history (#4045).
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    fillet::FilletDefinition,
    part::{PartFeatureDefinition, PartFeatureProgramError},
};

impl DocumentEditHistory {
    /// Add a fillet inside an existing transaction; example: `history.create_fillet(&token, "Fillet", definition)?`.
    pub fn create_fillet(
        &mut self,
        token: &TransactionToken,
        name: &str,
        definition: FilletDefinition,
    ) -> Result<FeatureId, EditError<HistoryError>> {
        self.edit(token, |snapshot, ids| {
            snapshot.create_fillet(name, definition, ids)
        })
    }
}

impl DocumentSnapshot {
    /// Append authored fillet inputs; example: `snapshot.create_fillet("Fillet", definition, ids)?`.
    pub fn create_fillet(
        &mut self,
        name: &str,
        definition: FilletDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let reads = definition.reads();
        let feature = AuthoredFeature::new(
            id,
            name,
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        );
        self.work_geometry
            .append(feature)
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))?;
        if let Some(states) = &mut self.model_states {
            states.register_feature(id.0);
        }
        Ok(id)
    }
}
