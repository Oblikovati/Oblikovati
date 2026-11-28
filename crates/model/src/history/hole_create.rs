// SPDX-License-Identifier: GPL-2.0-only
//! Create an authored Hole in the common part history (#4093).
use super::*;
use crate::feature::{FeatureId, hole::HoleDefinition, part::PartFeatureDefinition};

impl DocumentEditHistory {
    /// Add a Hole inside a transaction; example: `history.create_hole(&token, "Hole", definition)?`.
    pub fn create_hole(
        &mut self,
        token: &TransactionToken,
        name: &str,
        definition: HoleDefinition,
    ) -> Result<FeatureId, EditError<HistoryError>> {
        self.edit(token, |snapshot, ids| {
            snapshot.create_hole(name, definition, ids)
        })
    }
}
impl DocumentSnapshot {
    /// Append Hole inputs; example: `snapshot.create_hole("Hole", definition, ids)?`.
    pub fn create_hole(
        &mut self,
        name: &str,
        definition: HoleDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Hole(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Hole definition in place; example: `snapshot.edit_hole(id, definition)?`.
    pub fn edit_hole(
        &mut self,
        feature: FeatureId,
        definition: HoleDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Hole(Box::new(definition)),
                reads,
            )
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })
            .map(|_| ())
    }

    /// Append Slot inputs; example: `snapshot.create_slot("Slot1", definition, ids)?`.
    pub fn create_slot(
        &mut self,
        name: &str,
        definition: crate::feature::slot::SlotDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Slot(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Slot definition in place; example: `snapshot.edit_slot(id, definition)?`.
    pub fn edit_slot(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::slot::SlotDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Slot(Box::new(definition)),
                reads,
            )
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })
            .map(|_| ())
    }

    /// Append Thread inputs; example: `snapshot.create_thread("Thread1", definition, ids)?`.
    pub fn create_thread(
        &mut self,
        name: &str,
        definition: crate::feature::thread::ThreadDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Thread(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Thread definition in place; example: `snapshot.edit_thread(id, definition)?`.
    pub fn edit_thread(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::thread::ThreadDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Thread(Box::new(definition)),
                reads,
            )
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })
            .map(|_| ())
    }
}
