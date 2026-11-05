// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches};
use crate::sketch::{SketchConstraintId, SketchDimensionLimits, SketchError};
impl DocumentSketches {
    /// Change a dimension role and invalidate verification only when its equation membership changes.
    /// Example: `sketches.set_dimension_reference(sketch, dimension, true)?;`.
    pub fn set_dimension_reference(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        reference: bool,
    ) -> Result<(), SketchError> {
        let index = self.index(sketch)?;
        let placed = &mut self.sketches[index];
        let before = placed.geometry.dimension_is_reference(dimension)?;
        placed
            .geometry
            .set_dimension_reference(dimension, reference)?;
        if before != reference {
            placed.evaluation = None;
        }
        Ok(())
    }
    /// Set drive bounds while retaining unchanged numerical verification.
    /// Example: `sketches.set_dimension_limits(sketch, dimension, limits)?;`.
    pub fn set_dimension_limits(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        limits: SketchDimensionLimits,
    ) -> Result<(), SketchError> {
        let index = self.index(sketch)?;
        self.sketches[index]
            .geometry
            .set_dimension_limits(dimension, limits)
    }
}
