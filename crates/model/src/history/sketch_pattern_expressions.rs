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
    /// Edit a recorded pattern and its owned spacing expressions atomically.
    /// Example: `snapshot.edit_pattern_from_expressions(sketch, pattern, request, reservations)?;`.
    pub fn edit_pattern_from_expressions(
        &mut self,
        sketch: DocumentSketchId,
        pattern: SketchEntityId,
        request: SketchRectangularPatternRequest,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<crate::sketch::SketchRectangularPatternEdit, SketchCopyError> {
        let geometry = self
            .sketches
            .get(sketch)
            .map_err(SketchCopyError::Sketch)?
            .geometry();
        let old = geometry
            .rectangular_patterns()
            .iter()
            .find(|row| row.id == pattern)
            .ok_or_else(|| {
                SketchCopyError::Sketch(SketchError::Definition(format!(
                    "pattern {pattern:?}: expected a recorded rectangular pattern"
                )))
            })?;
        let spacing = old.definition.spacing;
        if spacing[0] == spacing[1] && request.spacing[0] != request.spacing[1] {
            return Err(SketchCopyError::Sketch(SketchError::Definition(format!(
                "pattern {pattern:?} shared spacing parameter {}: expected matching expressions",
                spacing[0]
            ))));
        }
        let mut candidate = self.clone();
        for (id, source) in spacing.into_iter().zip(&request.spacing) {
            candidate.update_pattern_spacing(id, source)?;
        }
        let definition = SketchRectangularPatternDefinition {
            seed: request.seed,
            counts: request.counts,
            directions: request.directions,
            spacing,
            fitted: request.fitted,
            suppressed: request.suppressed,
        };
        let edit = candidate.edit_rectangular_pattern(sketch, pattern, definition, reservations)?;
        *self = candidate;
        Ok(edit)
    }
    fn update_pattern_spacing(&mut self, id: u64, source: &str) -> Result<(), SketchCopyError> {
        let old = self.parameters.get(id).ok_or_else(|| {
            SketchCopyError::Sketch(SketchError::Definition(format!(
                "pattern spacing parameter {id}: expected an existing source"
            )))
        })?;
        if old.source == source {
            return Ok(());
        }
        self.parameters.set_scalar_expression(id, source, Dimension::Length)
            .map_err(|cause| SketchCopyError::Sketch(SketchError::Definition(format!("pattern spacing parameter {id} expression {source:?}: expected an editable Model length: {cause}"))))
    }
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
