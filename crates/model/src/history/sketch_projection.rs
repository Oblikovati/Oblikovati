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
                .effective_work_geometry()
                .evaluate_work_geometry(&snapshot.parameters),
        }
    }
}
impl SketchProjectionInputs for CurrentSketchProjectionInputs {
    fn points(&self, placed: &PlacedSketch) -> Result<Vec<SketchPoint2D>, SketchError> {
        BorrowedSketchProjectionInputs {
            datums: self.datums.as_ref(),
        }
        .points(placed)
    }
}
pub(super) struct BorrowedSketchProjectionInputs<'a> {
    datums: Result<&'a WorkGeometryEvaluation, &'a FeatureReplayError>,
}
impl<'a> BorrowedSketchProjectionInputs<'a> {
    pub(super) fn from_datums(datums: &'a WorkGeometryEvaluation) -> Self {
        Self { datums: Ok(datums) }
    }
}
impl SketchProjectionInputs for BorrowedSketchProjectionInputs<'_> {
    fn points(&self, placed: &PlacedSketch) -> Result<Vec<SketchPoint2D>, SketchError> {
        placed
            .point_projections()
            .iter()
            .map(|binding| self.binding_point(placed, *binding))
            .collect()
    }
}
impl BorrowedSketchProjectionInputs<'_> {
    fn binding_point(
        &self,
        placed: &PlacedSketch,
        binding: SketchPointProjection,
    ) -> Result<SketchPoint2D, SketchError> {
        let datums = self.datums.map_err(|cause| SketchError::ProjectionSource {
            point: binding.anchor,
            cause: Box::new(DocumentSketchPlacementError::Replay(cause.clone())),
        })?;
        projection_anchor(placed, datums, binding)
    }
}

impl DocumentSnapshot {
    /// Inspect sketch health with projection inputs from one part replay.
    /// Example: `snapshot.document_sketch_health_with_datums(id, &datums)?`.
    pub fn document_sketch_health_with_datums(
        &self,
        id: DocumentSketchId,
        datums: &WorkGeometryEvaluation,
    ) -> Result<crate::document_sketch::PlacedSketchHealth<'_>, SketchError> {
        let placed = self.sketches.get(id)?;
        let projections = self.resolve_sketch_projected_points_with_datums(id, datums);
        Ok(placed.health_with_projection_inputs(&self.parameters, &projections))
    }
    /// Resolve projection inputs from one ordered part replay.
    /// Example: `snapshot.resolve_sketch_projected_points_with_datums(id, &datums)?`.
    pub fn resolve_sketch_projected_points_with_datums(
        &self,
        id: DocumentSketchId,
        datums: &WorkGeometryEvaluation,
    ) -> Result<Vec<SketchPoint2D>, SketchError> {
        self.sketches
            .get(id)?
            .projection_inputs(&BorrowedSketchProjectionInputs::from_datums(datums))
    }
    /// Project a datum through the same part replay as the sketch placement.
    /// Example: `snapshot.project_datum_point_into_sketch_with_datums(id, source, &datums)?`.
    pub fn project_datum_point_into_sketch_with_datums(
        &self,
        sketch: DocumentSketchId,
        reference: WorkPointReference,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedSketchPointProjection, DocumentSketchPlacementError> {
        let placed = self
            .sketches
            .get(sketch)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        project_current_point(placed, datums, reference)
    }
    /// Inspect retained solve evidence against current parameters and projected anchors.
    /// Example: `snapshot.document_sketch_health(id)?` before displaying a health badge.
    pub fn document_sketch_health(
        &self,
        id: DocumentSketchId,
    ) -> Result<crate::document_sketch::PlacedSketchHealth<'_>, SketchError> {
        let placed = self.sketches.get(id)?;
        let projections = self.resolve_sketch_projected_points(id);
        Ok(placed.health_with_projection_inputs(&self.parameters, &projections))
    }
    /// Resolve saved anchors from the current immutable owner; example: `snapshot.resolve_sketch_projected_points(id)?`.
    pub fn resolve_sketch_projected_points(
        &self,
        id: DocumentSketchId,
    ) -> Result<Vec<SketchPoint2D>, SketchError> {
        let placed = self.sketches.get(id)?;
        placed.projection_inputs(&CurrentSketchProjectionInputs::new(self))
    }
    /// Resolve only selected projection anchors while retaining errors from those sources.
    /// Example: `snapshot.resolve_selected_sketch_projections(id, &anchors)?` for an isolated component.
    pub fn resolve_selected_sketch_projections(
        &self,
        sketch: DocumentSketchId,
        anchors: &[crate::sketch::SketchEntityId],
    ) -> Result<Vec<SketchPoint2D>, SketchError> {
        let placed = self.sketches.get(sketch)?;
        let current = CurrentSketchProjectionInputs::new(self);
        let inputs = BorrowedSketchProjectionInputs {
            datums: current.datums.as_ref(),
        };
        placed
            .point_projections()
            .iter()
            .filter(|binding| anchors.contains(&binding.anchor))
            .map(|binding| inputs.binding_point(placed, *binding))
            .collect()
    }
    /// Project a current datum point into a current sketch frame without requiring a solved sketch.
    /// Example: `snapshot.project_datum_point_into_sketch(id, WorkPointReference::Origin)?`.
    pub fn project_datum_point_into_sketch(
        &self,
        sketch: DocumentSketchId,
        reference: WorkPointReference,
    ) -> Result<ResolvedSketchPointProjection, DocumentSketchPlacementError> {
        // #3760: source, host plane and orientation must consume the same immutable replay.
        let datums = self
            .effective_work_geometry()
            .evaluate_work_geometry(&self.parameters)
            .map_err(DocumentSketchPlacementError::Replay)?;
        self.project_datum_point_into_sketch_with_datums(sketch, reference, &datums)
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

fn projection_anchor(
    placed: &PlacedSketch,
    datums: &WorkGeometryEvaluation,
    binding: SketchPointProjection,
) -> Result<SketchPoint2D, SketchError> {
    project_current_point(placed, datums, binding.reference)
        .map(|projection| SketchPoint2D {
            id: binding.anchor,
            position: projection.local_position,
        })
        .map_err(|cause| SketchError::ProjectionSource {
            point: binding.anchor,
            cause: Box::new(cause),
        })
}
impl DocumentSnapshot {
    pub(crate) fn projected_points_with_datums(
        &self,
        id: DocumentSketchId,
        datums: &WorkGeometryEvaluation,
    ) -> Result<Vec<SketchPoint2D>, SketchError> {
        let placed = self.sketches.get(id)?;
        placed.projection_inputs(&BorrowedSketchProjectionInputs { datums: Ok(datums) })
    }
}
