// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityEditError, SketchEntityId, SketchError, SketchLineSplitReport},
};
impl DocumentSnapshot {
    /// Split the first selected line after validating the complete selection; reservations survive later rejection.
    /// Example: `snapshot.split_sketch_line(sketch, &[line], [2.0, 0.0], reservations)?;`.
    pub fn split_sketch_line(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        pick: [f64; 2],
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<SketchLineSplitReport, SketchEntityEditError> {
        let source = self
            .sketches
            .get(sketch)
            .map_err(SketchEntityEditError::Sketch)?;
        let mut geometry = source.geometry().clone();
        let line = split_selection(&geometry, entities).map_err(SketchEntityEditError::Sketch)?;
        let report =
            geometry.split_line(line, pick, &mut || reservations.allocate_sketch_entity())?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchEntityEditError::Sketch)?;
        Ok(report)
    }
}
fn split_selection(
    geometry: &Sketch2D,
    selected: &[SketchEntityId],
) -> Result<SketchEntityId, SketchError> {
    for entity in selected {
        geometry.entity_kind(*entity)?;
    }
    selected.first().copied().ok_or_else(|| {
        SketchError::Definition(format!(
            "split selection {selected:?}: expected at least one owned entity with a line first"
        ))
    })
}
