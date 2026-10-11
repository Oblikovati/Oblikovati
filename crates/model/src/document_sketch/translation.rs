// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches};
use crate::sketch::{SketchEntityId, SketchError};
impl DocumentSketches {
    /// Publish a checked selection edit and invalidate numerical evidence for recompute.
    /// Example: `sketches.translate_entities(sketch, &[line], [2.0, 3.0])?;`.
    pub fn translate_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        let mut geometry = self.get(sketch)?.geometry().clone();
        let selected = geometry.translate_entities(entities, vector)?;
        self.replace_geometry(sketch, geometry)?;
        Ok(selected)
    }
}
