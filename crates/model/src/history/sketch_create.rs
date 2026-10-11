// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchPlacementError, DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::{
        DocumentSketchId, SketchOrientationDefinition, SketchPlacement, SketchPointProjection,
    },
    feature::work_geometry::WorkPointReference,
    identity::RecipeIdError,
    sketch::{Sketch2D, SketchError},
};
use std::{error::Error, fmt};

/// Authored creation inputs; the caller supplies application preferences and stable references.
#[derive(Clone, Debug)]
pub struct DocumentSketchCreation {
    pub name: String,
    pub placement: SketchPlacement,
    pub orientation: Option<SketchOrientationDefinition>,
    pub project_origin: bool,
}
/// Creation keeps identity exhaustion and current source failures available to the caller.
#[derive(Clone, Debug, PartialEq)]
pub enum DocumentSketchCreationError {
    Identity(RecipeIdError),
    Sketch(SketchError),
    Placement(Box<DocumentSketchPlacementError>),
}
impl DocumentSnapshot {
    /// Adopt a checked sketch and its optional associative origin atomically.
    /// Example: `snapshot.create_document_sketch(&request, reservations)?` inside an owner command.
    pub fn create_document_sketch(
        &mut self,
        request: &DocumentSketchCreation,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<DocumentSketchId, DocumentSketchCreationError> {
        let sketch = reservations
            .allocate_sketch()
            .map_err(DocumentSketchCreationError::Identity)?;
        let mut candidate = self.clone();
        candidate
            .insert_created_sketch(sketch, request)
            .map_err(DocumentSketchCreationError::Sketch)?;
        candidate
            .resolve_sketch_plane(sketch)
            .map_err(placement_failure)?;
        if request.project_origin {
            candidate.add_created_origin(sketch, reservations)?;
        }
        *self = candidate;
        Ok(sketch)
    }
    fn insert_created_sketch(
        &mut self,
        sketch: DocumentSketchId,
        request: &DocumentSketchCreation,
    ) -> Result<(), SketchError> {
        let geometry = Sketch2D::default();
        match request.placement {
            SketchPlacement::Fixed(plane) => {
                self.sketches.add(sketch, &request.name, plane, geometry)?
            }
            SketchPlacement::WorkPlane(feature) => {
                self.sketches
                    .add_on_work_plane(sketch, &request.name, feature, geometry)?
            }
        }
        self.sketches.set_orientation(sketch, request.orientation)
    }
    fn add_created_origin(
        &mut self,
        sketch: DocumentSketchId,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<(), DocumentSketchCreationError> {
        let reference = WorkPointReference::Origin;
        let projection = self
            .project_datum_point_into_sketch(sketch, reference)
            .map_err(placement_failure)?;
        let anchor = reservations
            .allocate_sketch_entity()
            .map_err(DocumentSketchCreationError::Identity)?;
        let mut geometry = Sketch2D::default();
        geometry
            .add_point(anchor, projection.local_position)
            .map_err(DocumentSketchCreationError::Sketch)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(DocumentSketchCreationError::Sketch)?;
        self.sketches
            .set_point_projections(sketch, vec![SketchPointProjection { anchor, reference }])
            .map_err(DocumentSketchCreationError::Sketch)
    }
}
fn placement_failure(cause: DocumentSketchPlacementError) -> DocumentSketchCreationError {
    DocumentSketchCreationError::Placement(Box::new(cause))
}
impl fmt::Display for DocumentSketchCreationError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Identity(cause) => write!(formatter, "sketch creation identity: {cause}"),
            Self::Sketch(cause) => write!(formatter, "sketch creation: {cause}"),
            Self::Placement(cause) => write!(formatter, "sketch creation placement: {cause}"),
        }
    }
}
impl Error for DocumentSketchCreationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Identity(cause) => Some(cause),
            Self::Sketch(cause) => Some(cause),
            Self::Placement(cause) => Some(cause.as_ref()),
        }
    }
}
