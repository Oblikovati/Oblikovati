// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityEditError, SketchEntityId, SketchLineEnd, SketchLineEndpointReport},
};
impl DocumentSnapshot {
    /// Replace one owned line end as seed geometry; history retains every reserved identity.
    /// Example: `snapshot.replace_sketch_line_endpoint(sketch, line, SketchLineEnd::Last, [5.0, 0.0], ids)?;`.
    pub fn replace_sketch_line_endpoint(
        &mut self,
        sketch: DocumentSketchId,
        line: SketchEntityId,
        end: SketchLineEnd,
        position: [f64; 2],
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<SketchLineEndpointReport, SketchEntityEditError> {
        let source = self
            .sketches
            .get(sketch)
            .map_err(SketchEntityEditError::Sketch)?;
        let mut geometry = source.geometry().clone();
        let report = geometry.replace_line_endpoint(line, end, position, &mut || {
            reservations.allocate_sketch_entity()
        })?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchEntityEditError::Sketch)?;
        Ok(report)
    }
}
