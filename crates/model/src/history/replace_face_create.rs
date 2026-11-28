// SPDX-License-Identifier: GPL-2.0-only
//! Create and edit authored Part Surface Replace Face features in document snapshots (#3918).
use super::*;
use crate::feature::{FeatureId, part::PartFeatureDefinition, replace_face::ReplaceFaceDefinition};

impl DocumentSnapshot {
    /// Append Replace Face inputs; example: `snapshot.create_replace_face("Replace Face1", definition, ids)?`.
    pub fn create_replace_face(
        &mut self,
        name: &str,
        definition: ReplaceFaceDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::ReplaceFace(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Replace Face definition in place; example: `snapshot.edit_replace_face(id, definition)?`.
    pub fn edit_replace_face(
        &mut self,
        feature: FeatureId,
        definition: ReplaceFaceDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::ReplaceFace(Box::new(definition)),
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
