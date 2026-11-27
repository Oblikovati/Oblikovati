// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches};
use crate::sketch::{SketchEntityId, SketchError, SketchSelectionTransform};
impl DocumentSketches {
    /// Publish a checked selection edit and invalidate numerical evidence for recompute.
    /// Example: `sketches.translate_entities(sketch, &[line], [2.0, 3.0])?;`.
    pub fn translate_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        self.transform_entities(sketch, entities, SketchSelectionTransform::Move(vector))
    }
    /// Adopt a rigid seed edit; example: `sketches.transform_entities(id, &entities, transform)?;`.
    pub fn transform_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        transform: SketchSelectionTransform,
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        let mut geometry = self.get(sketch)?.geometry().clone();
        let selected = geometry.transform_entities(entities, transform)?;
        self.replace_geometry(sketch, geometry)?;
        Ok(selected)
    }
}
