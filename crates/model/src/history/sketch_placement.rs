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
/// A local placement or verification decline retains the source failure.
#[derive(Clone, Debug, PartialEq)]
pub enum DocumentSketchPlacementError {
    Sketch(SketchError),
    Replay(FeatureReplayError),
    Plane {
        sketch: DocumentSketchId,
        cause: WorkPlaneError,
    },
    Verification(SketchReadinessError),
}
impl DocumentSnapshot {
    /// Resolve current placement from this immutable owner; example: `snapshot.resolve_sketch_plane(id)?`.
    pub fn resolve_sketch_plane(
        &self,
        id: DocumentSketchId,
    ) -> Result<SketchPlane, DocumentSketchPlacementError> {
        self.resolve_axis_inputs(id, None)
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
        let needs_axis = owned_axes
            && placed.orientation().is_some_and(|definition| {
                matches!(
                    definition.reference,
                    crate::document_sketch::SketchAxisReference::Feature(_)
                )
            });
        if matches!(placed.placement(), SketchPlacement::Fixed(_)) && !needs_axis {
            return Ok(None);
        }
        self.work_geometry
            .evaluate_work_geometry(&self.parameters)
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
        let sketch = self
            .sketches
            .get(id)
            .map_err(DocumentSketchPlacementError::Sketch)?;
        let verification = sketch
            .require_fully_constrained(&self.parameters)
            .map_err(DocumentSketchPlacementError::Verification)?;
        Ok(ResolvedDocumentSketch {
            sketch,
            plane,
            verification,
        })
    }
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
        }
    }
}
impl Error for DocumentSketchPlacementError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Sketch(cause) => Some(cause),
            Self::Replay(cause) => Some(cause),
            Self::Plane { cause, .. } => Some(cause),
            Self::Verification(cause) => Some(cause),
        }
    }
}
