// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    quantity::Dimension,
    sketch::{
        SketchCopyError, SketchEntityId, SketchError, SketchRectangularPattern,
        SketchRectangularPatternDefinition,
    },
};
use std::collections::BTreeSet;

/// Rectangular pattern inputs before their spacing expressions receive owned parameter IDs.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchRectangularPatternRequest {
    pub seed: Vec<SketchEntityId>,
    pub counts: [u32; 2],
    pub directions: [[f64; 2]; 2],
    pub spacing: [String; 2],
    pub fitted: bool,
    pub suppressed: BTreeSet<[u32; 2]>,
}
impl DocumentSnapshot {
    /// Create a pattern and its spacing expressions atomically.
    /// Example: `snapshot.create_pattern_from_expressions(sketch, request, reservations)?;`.
    pub fn create_pattern_from_expressions(
        &mut self,
        sketch: DocumentSketchId,
        request: SketchRectangularPatternRequest,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<SketchRectangularPattern, SketchCopyError> {
        self.sketches.get(sketch).map_err(SketchCopyError::Sketch)?;
        let mut candidate = self.clone();
        let spacing = [
            candidate.insert_pattern_spacing(sketch, &request.spacing[0], reservations)?,
            candidate.insert_pattern_spacing(sketch, &request.spacing[1], reservations)?,
        ];
        let definition = SketchRectangularPatternDefinition {
            seed: request.seed,
            counts: request.counts,
            directions: request.directions,
            spacing,
            fitted: request.fitted,
            suppressed: request.suppressed,
        };
        let pattern = candidate.create_rectangular_pattern(sketch, definition, reservations)?;
        *self = candidate;
        Ok(pattern)
    }
    fn insert_pattern_spacing(
        &mut self,
        sketch: DocumentSketchId,
        source: &str,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<u64, SketchCopyError> {
        let name = reservations
            .allocate_dimension_name(sketch, &self.parameters)
            .map_err(SketchCopyError::Identity)?;
        let id = reservations.allocate().map_err(SketchCopyError::Identity)?;
        self.parameters
            .insert_scalar_expression(id, &name, source, Dimension::Length)
            .map_err(|cause| {
                SketchCopyError::Sketch(SketchError::Definition(format!(
                    "pattern spacing {source:?}: expected a valid length expression: {cause}"
                )))
            })?;
        Ok(id)
    }
}
