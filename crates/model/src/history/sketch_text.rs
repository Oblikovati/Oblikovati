// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations, SketchPrimitiveCreationError};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityId, SketchError, SketchText2D},
};
use oblikovati_api::types::SketchTextStyle;

impl DocumentSnapshot {
    /// Add text and its defining point in one candidate, for example `snapshot.create_sketch_text(sketch, point, style, ids)?`.
    pub fn create_sketch_text(
        &mut self,
        sketch: DocumentSketchId,
        position: [f64; 2],
        style: SketchTextStyle,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchEntityId, SketchPrimitiveCreationError> {
        use SketchPrimitiveCreationError::Sketch;
        let mut geometry = self
            .sketches
            .get(sketch)
            .map_err(Sketch)?
            .geometry()
            .clone();
        let id = ids
            .allocate_sketch_entity()
            .map_err(SketchPrimitiveCreationError::Identity)?;
        let anchor = ids
            .allocate_sketch_entity()
            .map_err(SketchPrimitiveCreationError::Identity)?;
        geometry
            .add_defining_point(anchor, position)
            .map_err(Sketch)?;
        geometry
            .add_text(SketchText2D { id, anchor, style })
            .map_err(Sketch)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(Sketch)?;
        Ok(id)
    }

    /// Edit a text entity while preserving its anchor, for example `snapshot.edit_sketch_text(sketch, id, style)?`.
    pub fn edit_sketch_text(
        &mut self,
        sketch: DocumentSketchId,
        id: SketchEntityId,
        style: SketchTextStyle,
    ) -> Result<(), SketchError> {
        let mut geometry = self.sketches.get(sketch)?.geometry().clone();
        geometry.set_text_style(id, style)?;
        self.sketches.replace_geometry(sketch, geometry)
    }
}
