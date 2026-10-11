// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe};
use crate::{
    feature::{FeatureId, FeatureReadKey, work_geometry::WorkPointReference},
    sketch::{Sketch2D, SketchEntityId, SketchError, SketchPoint2D},
};
use std::collections::BTreeSet;

/// The reference drives this owned anchor; stored point coordinates are only a warm seed.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPointProjection {
    pub anchor: SketchEntityId,
    pub reference: WorkPointReference,
}
/// Current source resolution belongs to the immutable document, outside the numerical solver.
pub trait SketchProjectionInputs {
    /// Return one finite point per binding in binding order; example: `inputs.points(placed)?`.
    fn points(&self, sketch: &PlacedSketch) -> Result<Vec<SketchPoint2D>, SketchError>;
}
pub(super) struct UnavailableProjectionInputs;
impl SketchProjectionInputs for UnavailableProjectionInputs {
    fn points(&self, sketch: &PlacedSketch) -> Result<Vec<SketchPoint2D>, SketchError> {
        Err(SketchError::Definition(format!(
            "document sketch {:?} point projections {:?}: expected current document source resolution",
            sketch.id, sketch.point_projections
        )))
    }
}
impl PlacedSketch {
    /// Inspect authored associations without retaining their evaluated coordinates; example: `placed.point_projections()`.
    pub fn point_projections(&self) -> &[SketchPointProjection] {
        &self.point_projections
    }
    /// Find the existing anchor before reserving another identity; example: `placed.projected_point_for_source(WorkPointReference::Origin)`.
    pub fn projected_point_for_source(
        &self,
        reference: WorkPointReference,
    ) -> Option<SketchEntityId> {
        self.point_projections
            .iter()
            .find(|binding| binding.reference == reference)
            .map(|binding| binding.anchor)
    }
    pub(crate) fn projection_inputs(
        &self,
        inputs: &dyn SketchProjectionInputs,
    ) -> Result<Vec<SketchPoint2D>, SketchError> {
        if self.point_projections.is_empty() {
            return Ok(Vec::new());
        }
        let points = inputs.points(self)?;
        self.validate_projection_coordinates(&points)?;
        Ok(points)
    }
    fn validate_projection_coordinates(&self, points: &[SketchPoint2D]) -> Result<(), SketchError> {
        let valid = points.len() == self.point_projections.len()
            && points
                .iter()
                .zip(&self.point_projections)
                .all(|(point, binding)| {
                    point.id == binding.anchor
                        && point.position.iter().all(|value| value.is_finite())
                });
        if !valid {
            return Err(SketchError::Definition(format!(
                "document sketch {:?} projection inputs {points:?}: expected finite points in binding order {:?}",
                self.id, self.point_projections
            )));
        }
        Ok(())
    }
}
impl DocumentSketches {
    /// Set associations atomically after anchor checks; example: `sketches.set_point_projections(id, bindings)?`.
    pub fn set_point_projections(
        &mut self,
        id: DocumentSketchId,
        bindings: Vec<SketchPointProjection>,
    ) -> Result<(), SketchError> {
        let index = self.index(id)?;
        validate_bindings(&self.sketches[index].geometry, &bindings)?;
        if self.sketches[index].point_projections == bindings {
            return Ok(());
        }
        self.sketches[index].point_projections = bindings;
        self.sketches[index].evaluation = None;
        Ok(())
    }
}
impl SketchPointProjection {
    pub(super) fn read_key(self) -> Option<FeatureReadKey> {
        match self.reference {
            WorkPointReference::Origin => None,
            WorkPointReference::Feature(id) => Some(FeatureReadKey::Feature(id)),
        }
    }
}
pub(super) fn validate_bindings(
    geometry: &Sketch2D,
    bindings: &[SketchPointProjection],
) -> Result<(), SketchError> {
    let mut anchors = BTreeSet::new();
    let mut sources = BTreeSet::new();
    for binding in bindings {
        geometry.point(binding.anchor)?;
        let source = match binding.reference {
            WorkPointReference::Origin => None,
            WorkPointReference::Feature(id) => Some(id),
        };
        if source == Some(FeatureId(0))
            || !anchors.insert(binding.anchor)
            || !sources.insert(source)
        {
            return Err(SketchError::Definition(format!(
                "sketch point projection {binding:?}: expected a unique anchor and source with a nonzero feature ID"
            )));
        }
    }
    Ok(())
}
pub(super) fn restore_bindings(
    sketches: &mut DocumentSketches,
    recipe: &PlacedSketchRecipe,
    version: u32,
) -> Result<(), SketchError> {
    if !recipe.point_projections.is_empty() && version < 4 {
        return Err(SketchError::Definition(format!(
            "sketch {:?} point projections in recipe version {version}: expected version 4",
            recipe.id
        )));
    }
    sketches.set_point_projections(recipe.id, recipe.point_projections.clone())
}

#[cfg(test)]
#[path = "../../tests/document_sketch/point_projection.rs"]
mod tests;
