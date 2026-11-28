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

    /// Append Combine inputs; example: `snapshot.create_combine("Combine1", definition, ids)?`.
    pub fn create_combine(
        &mut self,
        name: &str,
        definition: crate::feature::combine::CombineDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Combine(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Combine definition in place; example: `snapshot.edit_combine(id, definition)?`.
    pub fn edit_combine(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::combine::CombineDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Combine(Box::new(definition)),
                reads,
            )
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })
            .map(|_| ())
    }

    /// Append Thicken/Offset inputs; example: `snapshot.create_thicken_offset("Thicken1", definition, ids)?`.
    pub fn create_thicken_offset(
        &mut self,
        name: &str,
        definition: crate::feature::thicken_offset::ThickenOffsetDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::ThickenOffset(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Thicken/Offset definition in place; example: `snapshot.edit_thicken_offset(id, definition)?`.
    pub fn edit_thicken_offset(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::thicken_offset::ThickenOffsetDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::ThickenOffset(Box::new(definition)),
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
