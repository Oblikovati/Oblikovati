// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchPlacementError, DocumentSnapshot};
use crate::{
    document_sketch::{
        DocumentSketchId, PlacedSketch, SketchPointProjection, SketchProjectionInputs,
    },
    feature::{
        FeatureReadFootprint, FeatureReadKey, FeatureReplayError,
        work_geometry::{WorkGeometryEvaluation, WorkPointReference},
    },
    sketch::{SketchError, SketchPoint2D},
};

/// Current analytic projection inputs; no solver seed or last-good source is stored.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedSketchPointProjection {
    pub reference: WorkPointReference,
    pub model_position: [f64; 3],
    pub local_position: [f64; 2],
    pub reads: FeatureReadFootprint,
}
pub(super) struct CurrentSketchProjectionInputs {
    datums: Result<WorkGeometryEvaluation, FeatureReplayError>,
}
impl CurrentSketchProjectionInputs {
    pub fn new(snapshot: &DocumentSnapshot) -> Self {
        Self {
            datums: snapshot
                .work_geometry
                .evaluate_work_geometry(&snapshot.parameters),
        }
    }
}
impl SketchProjectionInputs for CurrentSketchProjectionInputs {
    fn points(&self, placed: &PlacedSketch) -> Result<Vec<SketchPoint2D>, SketchError> {
        placed
            .point_projections()
            .iter()
            .map(|binding| self.binding_point(placed, *binding))
            .collect()
    }
}
impl CurrentSketchProjectionInputs {
    fn binding_point(
        &self,
        placed: &PlacedSketch,
        binding: SketchPointProjection,
    ) -> Result<SketchPoint2D, SketchError> {
        self.datums
            .as_ref()
            .map_err(|cause| DocumentSketchPlacementError::Replay(cause.clone()))
            .and_then(|datums| project_current_point(placed, datums, binding.reference))
            .map(|projection| SketchPoint2D {
                id: binding.anchor,
                position: projection.local_position,
            })
            .map_err(|cause| SketchError::ProjectionSource {
                point: binding.anchor,
                cause: Box::new(cause),
            })
    }
}

impl DocumentSnapshot {
    /// Resolve saved anchors from the current immutable owner; example: `snapshot.resolve_sketch_projected_points(id)?`.
    pub fn resolve_sketch_projected_points(
        &self,
        id: DocumentSketchId,
    ) -> Result<Vec<SketchPoint2D>, SketchError> {
        let placed = self.sketches.get(id)?;
        placed.projection_inputs(&CurrentSketchProjectionInputs::new(self))
    }
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
