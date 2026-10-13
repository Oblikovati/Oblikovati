// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchCopyError, SketchCopyOperation, SketchCopyReport, SketchEntityId},
};
impl DocumentSnapshot {
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
