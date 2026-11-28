// SPDX-License-Identifier: GPL-2.0-only
//! Create an authored Hole in the common part history (#4093).
use super::*;
use crate::feature::{FeatureId, hole::HoleDefinition, part::PartFeatureDefinition};

#[path = "ucs_create.rs"]
mod ucs_create;

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

    /// Append Draft inputs; example: `snapshot.create_draft("FaceDraft1", definition, ids)?`.
    pub fn create_draft(
        &mut self,
        name: &str,
        definition: crate::feature::draft::DraftDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Draft(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Draft definition in place; example: `snapshot.edit_draft(id, definition)?`.
    pub fn edit_draft(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::draft::DraftDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Draft(Box::new(definition)),
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

    /// Append Delete Face inputs; example: `snapshot.create_delete_face("Delete Face1", definition, ids)?`.
    pub fn create_delete_face(
        &mut self,
        name: &str,
        definition: crate::feature::delete_face::DeleteFaceDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::DeleteFace(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Delete Face definition in place; example: `snapshot.edit_delete_face(id, definition)?`.
    pub fn edit_delete_face(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::delete_face::DeleteFaceDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::DeleteFace(Box::new(definition)),
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

    /// Append Direct Edit inputs; example: `snapshot.create_direct_edit("Direct Edit1", definition, ids)?`.
    pub fn create_direct_edit(
        &mut self,
        name: &str,
        definition: crate::feature::direct_edit::DirectEditDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::DirectEdit(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Direct Edit definition in place; example: `snapshot.edit_direct_edit(id, definition)?`.
    pub fn edit_direct_edit(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::direct_edit::DirectEditDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::DirectEdit(Box::new(definition)),
                reads,
            )
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })
            .map(|_| ())
    }

    /// Append Split inputs; example: `snapshot.create_split("Split1", definition, ids)?`.
    pub fn create_split(
        &mut self,
        name: &str,
        definition: crate::feature::split::SplitDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Split(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Split definition in place; example: `snapshot.edit_split(id, definition)?`.
    pub fn edit_split(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::split::SplitDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Split(Box::new(definition)),
                reads,
            )
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })
            .map(|_| ())
    }

    /// Append Mark inputs; example: `snapshot.create_mark("Mark1", definition, ids)?`.
    pub fn create_mark(
        &mut self,
        name: &str,
        definition: crate::feature::mark::MarkDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads(self).map_err(|cause| {
            HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Mark {
                feature: FeatureId(0),
                cause,
            })
        })?;
        self.create_body_feature(
            name,
            PartFeatureDefinition::Mark(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Mark definition in place; example: `snapshot.edit_mark(id, definition)?`.
    pub fn edit_mark(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::mark::MarkDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads(self).map_err(|cause| {
            HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Mark {
                feature,
                cause,
            })
        })?;
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Mark(Box::new(definition)),
                reads,
            )
            .map_err(|cause| {
                HistoryError::PartProgram(crate::feature::part::PartFeatureProgramError::Sequence(
                    cause,
                ))
            })
            .map(|_| ())
    }

    /// Append Finish inputs; example: `snapshot.create_finish("Finish1", definition, ids)?`.
    pub fn create_finish(
        &mut self,
        name: &str,
        definition: crate::feature::finish::FinishDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Finish(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Finish definition in place; example: `snapshot.edit_finish(id, definition)?`.
    pub fn edit_finish(
        &mut self,
        feature: FeatureId,
        definition: crate::feature::finish::FinishDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Finish(Box::new(definition)),
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
