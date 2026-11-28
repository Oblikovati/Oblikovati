// SPDX-License-Identifier: GPL-2.0-only
//! Create and edit Surface Boundary Patch features in part history (#3918).
use super::super::*;
use crate::feature::{
    FeatureId, boundary_patch::BoundaryPatchDefinition, part::PartFeatureDefinition,
};

impl DocumentSnapshot {
    /// Append Boundary Patch inputs and hide consumed sketches; example: `snapshot.create_boundary_patch("Boundary Patch1", definition, ids)?`.
    pub fn create_boundary_patch(
        &mut self,
        name: &str,
        definition: BoundaryPatchDefinition,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, HistoryError> {
        let reads = definition.reads();
        let sketches = definition.referenced_sketches();
        let id = self.create_body_feature(
            name,
            PartFeatureDefinition::BoundaryPatch(Box::new(definition)),
            reads,
            ids,
        )?;
        for sketch in sketches {
            self.sketches
                .set_visible(sketch, false)
                .map_err(|_| HistoryError::UnreservedSketchId(sketch.0))?;
        }
        Ok(id)
    }

    /// Replace an existing Boundary Patch definition in place; example: `snapshot.edit_boundary_patch(id, definition)?`.
    pub fn edit_boundary_patch(
        &mut self,
        feature: FeatureId,
        definition: BoundaryPatchDefinition,
    ) -> Result<(), HistoryError> {
        let reads = definition.reads();
        self.work_geometry
            .replace(
                feature,
                PartFeatureDefinition::BoundaryPatch(Box::new(definition)),
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
