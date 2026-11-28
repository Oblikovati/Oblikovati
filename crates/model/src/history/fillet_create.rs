// SPDX-License-Identifier: GPL-2.0-only
//! Create authored fillet inputs in the common Part history (#4045).
use super::*;
use crate::feature::{
    AuthoredFeature, FeatureId,
    chamfer::ChamferDefinition,
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
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
            ids,
        )
    }
    /// Replace an existing fillet definition in-place; example: `snapshot.edit_fillet(id, definition)?`.
    pub fn edit_fillet(
        &mut self,
        id: FeatureId,
        definition: FilletDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                id,
                PartFeatureDefinition::Fillet(Box::new(definition)),
                reads,
            )
            .map(|_| ())
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))
    }
    /// Append authored chamfer inputs; example: `snapshot.create_chamfer("Chamfer", definition, ids)?`.
    pub fn create_chamfer(
        &mut self,
        name: &str,
        definition: ChamferDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Chamfer(Box::new(definition)),
            reads,
            ids,
        )
    }
    /// Replace an existing chamfer definition in-place; example: `snapshot.edit_chamfer(id, definition)?`.
    pub fn edit_chamfer(
        &mut self,
        id: FeatureId,
        definition: ChamferDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                id,
                PartFeatureDefinition::Chamfer(Box::new(definition)),
                reads,
            )
            .map(|_| ())
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))
    }
    pub(super) fn create_body_feature(
        &mut self,
        name: &str,
        definition: PartFeatureDefinition,
        reads: crate::feature::FeatureReadFootprint,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let id = ids.allocate_feature().map_err(HistoryError::Identity)?;
        let feature = AuthoredFeature::new(id, name, definition, reads);
        self.work_geometry
            .append(feature)
            .map_err(|cause| HistoryError::PartProgram(PartFeatureProgramError::Sequence(cause)))?;
        if let Some(states) = &mut self.model_states {
            states.register_feature(id.0);
        }
        Ok(id)
    }
}
