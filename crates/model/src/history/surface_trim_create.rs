// SPDX-License-Identifier: GPL-2.0-only
//! Append and edit Surface Trim features on a `DocumentSnapshot` (#3918).
use super::{DocumentSnapshot, HistoryError, RecipeReservations};
use crate::feature::{FeatureId, part::PartFeatureDefinition, surface_trim::SurfaceTrimDefinition};

impl DocumentSnapshot {
    /// Append Surface Trim inputs; example: `snapshot.create_surface_trim("Trim1", definition, ids)?`.
    pub fn create_surface_trim(
        &mut self,
        name: &str,
        definition: SurfaceTrimDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        self.create_body_feature(
            name,
            PartFeatureDefinition::SurfaceTrim(Box::new(definition)),
            reads,
            ids,
        )
    }

    /// Replace an existing Surface Trim definition in place; example: `snapshot.edit_surface_trim(id, definition)?`.
    pub fn edit_surface_trim(
        &mut self,
        feature: FeatureId,
        definition: SurfaceTrimDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::SurfaceTrim(Box::new(definition)),
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
