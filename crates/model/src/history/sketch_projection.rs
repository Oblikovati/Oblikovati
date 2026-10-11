// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchPlacementError, DocumentSnapshot};
use crate::{
    document_sketch::{DocumentSketchId, PlacedSketch},
    feature::{
        FeatureReadFootprint, FeatureReadKey,
        work_geometry::{WorkGeometryEvaluation, WorkPointReference},
    },
};

/// Current analytic projection inputs; no solver seed or last-good source is stored.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedSketchPointProjection {
    pub reference: WorkPointReference,
    pub model_position: [f64; 3],
    pub local_position: [f64; 2],
    pub reads: FeatureReadFootprint,
}

impl DocumentSnapshot {
    /// Project a current datum point into a current sketch frame without requiring a solved sketch.
    /// Example: `snapshot.project_datum_point_into_sketch(id, WorkPointReference::Origin)?`.
    pub fn project_datum_point_into_sketch(
        &self,
        sketch: DocumentSketchId,
        reference: WorkPointReference,
    ) -> Result<ResolvedSketchPointProjection, DocumentSketchPlacementError> {
        let placed = self
            .sketches
            .get(sketch)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        // #3760: source, host plane and orientation must consume the same immutable replay.
        let datums = self
            .work_geometry
            .evaluate_work_geometry(&self.parameters)
            .map_err(DocumentSketchPlacementError::Replay)?;
        project_current_point(placed, &datums, reference)
    }
}

fn project_current_point(
    placed: &PlacedSketch,
    datums: &WorkGeometryEvaluation,
    reference: WorkPointReference,
) -> Result<ResolvedSketchPointProjection, DocumentSketchPlacementError> {
    let sketch = placed.id();
    let plane = placed
        .plane_with_axes(Some(datums), datums)
        .map_err(|cause| DocumentSketchPlacementError::Plane { sketch, cause })?;
    let model_position = resolve_projection_source(sketch, reference, datums)?;
    let local_position = plane
        .to_sketch(model_position)
        .map_err(DocumentSketchPlacementError::Sketch)?;
    Ok(ResolvedSketchPointProjection {
        reference,
        model_position,
        local_position,
        reads: projection_reads(placed, reference),
    })
}

fn resolve_projection_source(
    sketch: DocumentSketchId,
    reference: WorkPointReference,
    datums: &WorkGeometryEvaluation,
) -> Result<[f64; 3], DocumentSketchPlacementError> {
    reference
        .position(datums)
        .map_err(|cause| DocumentSketchPlacementError::Point {
            sketch,
            reference,
            cause: Box::new(cause),
        })
}

fn projection_reads(placed: &PlacedSketch, reference: WorkPointReference) -> FeatureReadFootprint {
    let mut keys = placed.placement_read_keys();
    keys.push(FeatureReadKey::Sketch(placed.id().0));
    if let WorkPointReference::Feature(feature) = reference {
        keys.push(FeatureReadKey::Feature(feature));
    }
    FeatureReadFootprint::new(keys, false)
}
