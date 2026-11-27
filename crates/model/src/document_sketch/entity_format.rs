// SPDX-License-Identifier: GPL-2.0-only
//! Edit presentation without invalidating geometric solve evidence.
use super::{DocumentSketchId, DocumentSketches, SketchError};
use crate::sketch::SketchEntityId;

impl DocumentSketches {
    /// Set one member format while retaining the solve; example: `sketches.set_entity_format(sketch, line, format)?`.
    pub fn set_entity_format(
        &mut self,
        sketch: DocumentSketchId,
        entity: SketchEntityId,
        format: oblikovati_api::types::SketchEntityFormat,
    ) -> Result<(), SketchError> {
        let index = self.index(sketch)?;
        self.sketches[index]
            .geometry
            .set_entity_format(entity, format)
    }
}
