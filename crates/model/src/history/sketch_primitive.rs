// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    identity::RecipeIdError,
    sketch::{Sketch2D, SketchEntityId, SketchError, SketchPointInput, SketchPrimitiveCreation},
};
use std::{error::Error, fmt};

/// Created primitive and its defining points in constructor order; shared points retain their IDs.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchPrimitiveCreationReport {
    pub sketch: DocumentSketchId,
    pub entity: SketchEntityId,
    pub points: Vec<SketchEntityId>,
    pub created: Vec<SketchEntityId>,
}
/// Keep geometry declines distinct from exhausted permanent identity reservations.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchPrimitiveCreationError {
    Sketch(SketchError),
    Identity(RecipeIdError),
}
impl DocumentSnapshot {
    /// Append an owned primitive atomically; caller owns permanent identity reservations.
    /// Example: `snapshot.create_sketch_primitive(sketch, &input, reservations)?;` inside a command.
    pub fn create_sketch_primitive(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPrimitiveCreation,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<SketchPrimitiveCreationReport, SketchPrimitiveCreationError> {
        let mut geometry = self
            .sketches
            .get(sketch)
            .map_err(SketchPrimitiveCreationError::Sketch)?
            .geometry()
            .clone();
        input
            .validate(&geometry)
            .map_err(SketchPrimitiveCreationError::Sketch)?;
        let report = append_primitive(&mut geometry, sketch, input, reservations)?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchPrimitiveCreationError::Sketch)?;
        Ok(report)
    }
}
pub(super) fn append_primitive(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    input: &SketchPrimitiveCreation,
    reservations: &mut RecipeReservations<'_>,
) -> Result<SketchPrimitiveCreationReport, SketchPrimitiveCreationError> {
    let mut points = Vec::new();
    let mut created = Vec::new();
    for point in input.defining_points() {
        points.push(append_point(geometry, point, reservations, &mut created)?);
    }
    let entity = primitive_identity(input, &points, reservations, &mut created)?;
    input
        .append_curve(geometry, entity, &points)
        .map_err(SketchPrimitiveCreationError::Sketch)?;
    Ok(SketchPrimitiveCreationReport {
        sketch,
        entity,
        points,
        created,
    })
}
fn primitive_identity(
    input: &SketchPrimitiveCreation,
    points: &[SketchEntityId],
    reservations: &mut RecipeReservations<'_>,
    created: &mut Vec<SketchEntityId>,
) -> Result<SketchEntityId, SketchPrimitiveCreationError> {
    if matches!(input, SketchPrimitiveCreation::Point { .. }) {
        return Ok(points[0]);
    }
    reserve_created(reservations, created)
}
fn append_point(
    geometry: &mut Sketch2D,
    point: SketchPointInput,
    reservations: &mut RecipeReservations<'_>,
    created: &mut Vec<SketchEntityId>,
) -> Result<SketchEntityId, SketchPrimitiveCreationError> {
    match point {
        SketchPointInput::Owned(id) => Ok(id),
        SketchPointInput::New(position) => {
            let id = reserve_created(reservations, created)?;
            geometry
                .add_point(id, position)
                .map_err(SketchPrimitiveCreationError::Sketch)?;
            Ok(id)
        }
    }
}
fn reserve_created(
    reservations: &mut RecipeReservations<'_>,
    created: &mut Vec<SketchEntityId>,
) -> Result<SketchEntityId, SketchPrimitiveCreationError> {
    let id = reservations
        .allocate_sketch_entity()
        .map_err(SketchPrimitiveCreationError::Identity)?;
    created.push(id);
    Ok(id)
}
impl fmt::Display for SketchPrimitiveCreationError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Sketch(cause) => write!(formatter, "sketch primitive creation: {cause}"),
            Self::Identity(cause) => write!(formatter, "sketch primitive identity: {cause}"),
        }
    }
}
impl Error for SketchPrimitiveCreationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Sketch(cause) => Some(cause),
            Self::Identity(cause) => Some(cause),
        }
    }
}
