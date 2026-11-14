// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches};
use crate::sketch::{SketchCurveFormat, SketchCurveFormatChange, SketchEntityId, SketchError};
impl DocumentSketches {
    /// Convert roles without changing numerical verification or identity.
    /// Example: `sketches.toggle_curve_format(sketch, &[line], SketchCurveFormat::Centerline)?;`.
    pub fn toggle_curve_format(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        format: SketchCurveFormat,
    ) -> Result<SketchCurveFormatChange, SketchError> {
        let index = self.index(sketch)?;
        self.sketches[index]
            .geometry
            .toggle_curve_format(entities, format)
    }
}
