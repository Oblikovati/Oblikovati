// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::{
    document_sketch::{
        DocumentSketchId, PlacedSketch, SketchAxisResolver, SketchPlacement, SketchReadinessError,
        SketchSolveDiagnostics,
    },
    feature::{
        FeatureReplayError,
        work_plane::{WorkPlaneError, WorkPlaneEvaluation, WorkPlaneResolver},
    },
    sketch::{SketchError, SketchPlane},
};
use std::{error::Error, fmt};

/// A checked view borrows its immutable document; no derived frame is saved in history.
#[derive(Debug)]
pub struct ResolvedDocumentSketch<'a> {
    pub sketch: &'a PlacedSketch,
    pub plane: SketchPlane,
    pub verification: &'a SketchSolveDiagnostics,
}
/// Current concrete datums bound to an immutable snapshot for repeated frame queries.
/// Example: `let frames = snapshot.bind_sketch_frames(&datums)?; frames.resolve(id)?;`.
pub struct BoundSketchFrames<'a> {
    snapshot: &'a DocumentSnapshot,
    datums: &'a WorkPlaneEvaluation,
}
impl BoundSketchFrames<'_> {
    /// Resolve one frame without repeating the model comparison; example: `frames.resolve(id)?`.
    pub fn resolve(
        &self,
        id: DocumentSketchId,
    ) -> Result<SketchPlane, DocumentSketchPlacementError> {
        self.snapshot
            .resolve_sketch_plane_with_replay_datums(id, self.datums)
    }
}
enum RequiredSolve {
    FullyConstrained,
    Converged,
}
/// A local placement or verification decline retains the source failure.
#[derive(Clone, Debug, PartialEq)]
pub enum DocumentSketchPlacementError {
    Sketch(SketchError),
    Replay(FeatureReplayError),
    Plane {
        sketch: DocumentSketchId,
        cause: WorkPlaneError,
    },
    Point {
        sketch: DocumentSketchId,
        reference: crate::feature::work_geometry::WorkPointReference,
        cause: Box<WorkPlaneError>,
    },
    Verification(SketchReadinessError),
}
impl DocumentSnapshot {
    /// Verify concrete datums once for repeated queries; example: `snapshot.bind_sketch_frames(&datums)?`.
    pub fn bind_sketch_frames<'a>(
        &'a self,
        datums: &'a WorkPlaneEvaluation,
    ) -> Result<BoundSketchFrames<'a>, SketchError> {
        datums.require_model_source(self)?;
        Ok(BoundSketchFrames {
            snapshot: self,
            datums,
        })
    }
    /// Resolve a sketch against one ordered part replay.
    /// Example: `snapshot.resolve_sketch_plane_with_datums(id, &datums)?`.
    pub fn resolve_sketch_plane_with_datums(
        &self,
        id: DocumentSketchId,
        datums: &WorkPlaneEvaluation,
    ) -> Result<SketchPlane, DocumentSketchPlacementError> {
        if self
            .sketch_plane_needs_work_geometry(id)
            .map_err(DocumentSketchPlacementError::Sketch)?
        {
            datums
                .require_frame_source(self, id)
                .map_err(DocumentSketchPlacementError::Sketch)?;
        }
        self.resolve_sketch_plane_with_replay_datums(id, datums)
    }
    pub(crate) fn resolve_sketch_plane_with_replay_datums(
        &self,
        id: DocumentSketchId,
        datums: &WorkPlaneEvaluation,
    ) -> Result<SketchPlane, DocumentSketchPlacementError> {
        let placed = self
            .sketches
            .get(id)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        placed
            .plane_with_axes(Some(datums), datums)
            .map_err(|cause| DocumentSketchPlacementError::Plane { sketch: id, cause })
    }
    /// Resolve current placement from this immutable owner; example: `snapshot.resolve_sketch_plane(id)?`.
    pub fn resolve_sketch_plane(
        &self,
        id: DocumentSketchId,
    ) -> Result<SketchPlane, DocumentSketchPlacementError> {
        self.resolve_axis_inputs(id, None)
    }
    /// Check whether resolving a sketch needs feature-derived frames.
    /// Example: `snapshot.sketch_plane_needs_work_geometry(id)?` before ordered replay.
    pub fn sketch_plane_needs_work_geometry(
        &self,
        id: DocumentSketchId,
    ) -> Result<bool, SketchError> {
        Ok(placement_needs_datums(self.sketches.get(id)?, true))
    }
    /// Check the placement and point reads of a sketch summary.
    /// Example: `snapshot.sketch_inspection_needs_work_geometry(id)?` before fetching datums.
    pub fn sketch_inspection_needs_work_geometry(
        &self,
        id: DocumentSketchId,
    ) -> Result<bool, SketchError> {
        let sketch = self.sketches.get(id)?;
        Ok(placement_needs_datums(sketch, true) || !sketch.point_projections().is_empty())
    }
    /// Resolve placement with the current injected axis catalogue; example: `snapshot.resolve_sketch_plane_with_axes(id, &axes)?`.
    pub fn resolve_sketch_plane_with_axes(
        &self,
        id: DocumentSketchId,
        axes: &dyn SketchAxisResolver,
    ) -> Result<SketchPlane, DocumentSketchPlacementError> {
        self.resolve_axis_inputs(id, Some(axes))
    }
    fn resolve_axis_inputs(
        &self,
        id: DocumentSketchId,
        injected: Option<&dyn SketchAxisResolver>,
    ) -> Result<SketchPlane, DocumentSketchPlacementError> {
        let placed = self
            .sketches
            .get(id)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        let datums = self.placement_frames(placed, injected.is_none())?;
        let planes = datums.as_ref().map(|datum| datum as &dyn WorkPlaneResolver);
        let owned = datums
            .as_ref()
            .map(|datum| datum as &dyn SketchAxisResolver);
        let axes = injected
            .or(owned)
            .unwrap_or(&crate::document_sketch::UnavailableSketchAxes);
        let result = placed.plane_with_axes(planes, axes);
        result.map_err(|cause| DocumentSketchPlacementError::Plane { sketch: id, cause })
    }
    fn placement_frames(
        &self,
        placed: &PlacedSketch,
        owned_axes: bool,
    ) -> Result<Option<WorkPlaneEvaluation>, DocumentSketchPlacementError> {
        if !placement_needs_datums(placed, owned_axes) {
            return Ok(None);
        }
        self.effective_work_geometry()
            .evaluate_work_geometry_unbound(&self.parameters)
            .map(Some)
            .map_err(DocumentSketchPlacementError::Replay)
    }

    /// Require a current available frame and a fully constrained solve before feature use.
    /// Example: `let view = snapshot.require_resolved_sketch(id)?; view.plane.to_model(point)?;`.
    pub fn require_resolved_sketch(
        &self,
        id: DocumentSketchId,
    ) -> Result<ResolvedDocumentSketch<'_>, DocumentSketchPlacementError> {
        let plane = self.resolve_sketch_plane(id)?;
        let points = self
            .resolve_sketch_projected_points(id)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        self.verified_sketch(id, plane, points, RequiredSolve::FullyConstrained)
    }
    /// Resolve current solved geometry for profile use, including free variables.
    /// Example: `snapshot.require_converged_resolved_sketch(id)?` before region discovery.
    pub fn require_converged_resolved_sketch(
        &self,
        id: DocumentSketchId,
    ) -> Result<ResolvedDocumentSketch<'_>, DocumentSketchPlacementError> {
        let plane = self.resolve_sketch_plane(id)?;
        let points = self
            .resolve_sketch_projected_points(id)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        self.verified_sketch(id, plane, points, RequiredSolve::Converged)
    }
    /// Resolve converged geometry with the current named Part placement.
    /// Example: `snapshot.require_converged_resolved_sketch_with_datums(id, datums)?`.
    pub fn require_converged_resolved_sketch_with_datums(
        &self,
        id: DocumentSketchId,
        datums: &WorkPlaneEvaluation,
    ) -> Result<ResolvedDocumentSketch<'_>, DocumentSketchPlacementError> {
        datums
            .require_frame_source(self, id)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        self.require_converged_sketch_with_datums(id, datums)
    }
    pub(crate) fn require_converged_sketch_with_datums(
        &self,
        id: DocumentSketchId,
        datums: &WorkPlaneEvaluation,
    ) -> Result<ResolvedDocumentSketch<'_>, DocumentSketchPlacementError> {
        let plane = self.resolve_sketch_plane_with_replay_datums(id, datums)?;
        let points = self
            .projected_points_with_datums(id, datums)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        self.verified_sketch(id, plane, points, RequiredSolve::Converged)
    }
    fn verified_sketch(
        &self,
        id: DocumentSketchId,
        plane: SketchPlane,
        points: Vec<crate::sketch::SketchPoint2D>,
        requirement: RequiredSolve,
    ) -> Result<ResolvedDocumentSketch<'_>, DocumentSketchPlacementError> {
        let sketch = self
            .sketches
            .get(id)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        let projections = Ok(points);
        let verification = match requirement {
            RequiredSolve::FullyConstrained => sketch
                .require_fully_constrained_with_projection_inputs(&self.parameters, &projections),
            RequiredSolve::Converged => {
                sketch.require_converged_with_projection_inputs(&self.parameters, &projections)
            }
        }
        .map_err(DocumentSketchPlacementError::Verification)?;
        Ok(ResolvedDocumentSketch {
            sketch,
            plane,
            verification,
        })
    }
}

fn placement_needs_datums(placed: &PlacedSketch, owned_axes: bool) -> bool {
    let needs_axis = owned_axes
        && placed.orientation().is_some_and(|definition| {
            matches!(
                definition.reference,
                crate::document_sketch::SketchAxisReference::Feature(_)
            )
        });
    needs_axis
        || (owned_axes
            && placed
                .frame_edit()
                .is_some_and(|edit| edit.requires_datums()))
        || !matches!(
            placed.placement(),
            SketchPlacement::Fixed(_) | SketchPlacement::OriginPlane(_)
        )
}
impl fmt::Display for DocumentSketchPlacementError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Sketch(cause) => cause.fmt(output),
            Self::Replay(cause) => cause.fmt(output),
            Self::Plane { sketch, cause } => {
                write!(output, "document sketch {sketch:?} placement: {cause}")
            }
            Self::Verification(cause) => cause.fmt(output),
            Self::Point {
                sketch,
                reference,
                cause,
            } => {
                write!(
                    output,
                    "document sketch {sketch:?} projected point {reference:?}: {cause}"
                )
            }
        }
    }
}
impl Error for DocumentSketchPlacementError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Sketch(cause) => Some(cause),
            Self::Replay(cause) => Some(cause),
            Self::Plane { cause, .. } => Some(cause),
            Self::Point { cause, .. } => Some(cause.as_ref()),
            Self::Verification(cause) => Some(cause),
        }
    }
}
