// SPDX-License-Identifier: GPL-2.0-only
//! DocumentSnapshot helpers for creating and editing Part Surface Ruled Surface features (#3918).
use crate::feature::{
    FeatureId, part::PartFeatureDefinition, ruled_surface::RuledSurfaceDefinition,
};
use crate::history::{DocumentSnapshot, HistoryError, RecipeReservations};

impl DocumentSnapshot {
    /// Append Ruled Surface inputs; example: `snapshot.create_ruled_surface("Ruled Surface1", definition, ids)?`.
    pub fn create_ruled_surface(
        &mut self,
        name: &str,
        definition: RuledSurfaceDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::RuledSurface(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Ruled Surface definition in place; example: `snapshot.edit_ruled_surface(id, definition)?`.
    pub fn edit_ruled_surface(
        &mut self,
        feature: FeatureId,
        definition: RuledSurfaceDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::RuledSurface(Box::new(definition)),
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
