// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchCopyError, SketchCopyReport, SketchEntityId},
};
impl DocumentSnapshot {
    /// Copy an owned selection; reservations stay consumed if a later command stage declines.
    /// Example: `snapshot.copy_sketch_entities(sketch, &[line], [2.0, 0.0], reservations)?;`.
    pub fn copy_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<SketchCopyReport, SketchCopyError> {
        let source = self.sketches.get(sketch).map_err(SketchCopyError::Sketch)?;
        let mut geometry = source.geometry().clone();
        let report = geometry.copy_entities(entities, vector, &mut || {
            reservations.allocate_sketch_entity()
        })?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchCopyError::Sketch)?;
        Ok(report)
    }
}
