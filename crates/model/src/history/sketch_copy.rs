// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchCopyError, SketchCopyOperation, SketchCopyReport, SketchEntityId},
};
impl DocumentSnapshot {
    /// Create a recorded rectangular pattern with permanently reserved identities.
    /// Example: `snapshot.create_rectangular_pattern(sketch, definition, reservations)?;`.
    pub fn create_rectangular_pattern(
        &mut self,
        sketch: DocumentSketchId,
        definition: crate::sketch::SketchRectangularPatternDefinition,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<crate::sketch::SketchRectangularPattern, SketchCopyError> {
        let source = self.sketches.get(sketch).map_err(SketchCopyError::Sketch)?;
        let mut geometry = source.geometry().clone();
        let id = reservations
            .allocate_sketch_entity()
            .map_err(SketchCopyError::Identity)?;
        let pattern =
            geometry.create_rectangular_pattern(id, definition, &self.parameters, &mut || {
                reservations.allocate_sketch_entity()
            })?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchCopyError::Sketch)?;
        Ok(pattern)
    }
    /// Copy linked members whose offsets are evaluated from document parameters.
    /// Example: `snapshot.copy_parametric_translations(sketch, &[circle], &[terms], reservations)?;`.
    pub fn copy_parametric_translations(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        occurrences: &[Vec<crate::sketch::SketchPatternOffsetTerm>],
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<Vec<SketchCopyReport>, SketchCopyError> {
        let source = self.sketches.get(sketch).map_err(SketchCopyError::Sketch)?;
        let mut geometry = source.geometry().clone();
        let reports = geometry.copy_parametric_translations(
            entities,
            occurrences,
            &self.parameters,
            &mut || reservations.allocate_sketch_entity(),
        )?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchCopyError::Sketch)?;
        Ok(reports)
    }
    /// Copy and link translated sketch members as one document edit.
    /// Example: `snapshot.copy_associative_translations(sketch, &[circle], &[[10.0, 0.0]], reservations)?;`.
    pub fn copy_associative_translations(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        offsets: &[[f64; 2]],
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<Vec<SketchCopyReport>, SketchCopyError> {
        let source = self.sketches.get(sketch).map_err(SketchCopyError::Sketch)?;
        let mut geometry = source.geometry().clone();
        let reports = geometry.copy_associative_translations(entities, offsets, &mut || {
            reservations.allocate_sketch_entity()
        })?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchCopyError::Sketch)?;
        Ok(reports)
    }
    /// Copy multiple instances as one snapshot edit; failed edits retain reserved identities.
    /// Example: `snapshot.copy_selected_sketch_entities_batch(sketch, &[line], &[operation], reservations)?;`.
    pub fn copy_selected_sketch_entities_batch(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        operations: &[SketchCopyOperation],
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<Vec<SketchCopyReport>, SketchCopyError> {
        let source = self.sketches.get(sketch).map_err(SketchCopyError::Sketch)?;
        let mut geometry = source.geometry().clone();
        let reports = geometry.copy_selected_entities_batch(entities, operations, &mut || {
            reservations.allocate_sketch_entity()
        })?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchCopyError::Sketch)?;
        Ok(reports)
    }
    /// Copy an owned selection; reservations stay consumed if a later command stage declines. Resolve the operation in its current owner; example: `snapshot.copy_selected_sketch_entities(sketch, &[arc], SketchCopyOperation::Mirror(line), reservations)?;`.
    pub fn copy_selected_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        operation: SketchCopyOperation,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<SketchCopyReport, SketchCopyError> {
        let source = self.sketches.get(sketch).map_err(SketchCopyError::Sketch)?;
        let mut geometry = source.geometry().clone();
        let report = geometry.copy_selected_entities(entities, operation, &mut || {
            reservations.allocate_sketch_entity()
        })?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchCopyError::Sketch)?;
        Ok(report)
    }
}
