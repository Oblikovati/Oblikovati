// SPDX-License-Identifier: GPL-2.0-only
//! Create and edit authored Part Surface Stitch features in document history (#3918).
use super::*;
use crate::feature::{FeatureId, part::PartFeatureDefinition, stitch::StitchDefinition};

impl DocumentSnapshot {
    /// Append Stitch inputs; example: `snapshot.create_stitch("Stitch Surface1", definition, ids)?`.
    pub fn create_stitch(
        &mut self,
        name: &str,
        definition: StitchDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::Stitch(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Stitch definition in place; example: `snapshot.edit_stitch(id, definition)?`.
    pub fn edit_stitch(
        &mut self,
        feature: FeatureId,
        definition: StitchDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::Stitch(Box::new(definition)),
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
