// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches};
use crate::sketch::{SketchConstraintId, SketchError};
impl DocumentSketches {
    /// Move an owned dimension label while retaining numerical verification.
    /// Example: `sketches.set_dimension_text_point(sketch, dimension, [2.0, 3.0])?;`.
    pub fn set_dimension_text_point(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        position: [f64; 2],
    ) -> Result<(), SketchError> {
        let index = self.index(sketch)?;
        self.sketches[index]
            .geometry
            .set_dimension_text_point(dimension, position)
    }
}
