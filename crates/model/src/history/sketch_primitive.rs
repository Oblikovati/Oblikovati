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
    let mut created = Vec::new();
    let points = append_defining_points(geometry, input, reservations, &mut created)?;
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
type PrimitivePointAdoption =
    fn(&mut Sketch2D, SketchEntityId, [f64; 2]) -> Result<(), SketchError>;
fn append_defining_points(
    geometry: &mut Sketch2D,
    input: &SketchPrimitiveCreation,
    reservations: &mut RecipeReservations<'_>,
    created: &mut Vec<SketchEntityId>,
) -> Result<Vec<SketchEntityId>, SketchPrimitiveCreationError> {
    let adopt: PrimitivePointAdoption = match input {
        SketchPrimitiveCreation::Point { .. } => Sketch2D::add_point,
        _ => Sketch2D::add_defining_point,
    };
    input
        .defining_points()
        .into_iter()
        .map(|point| append_point(geometry, point, reservations, created, adopt))
        .collect()
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
pub(super) fn append_point(
    geometry: &mut Sketch2D,
    point: SketchPointInput,
    reservations: &mut RecipeReservations<'_>,
    created: &mut Vec<SketchEntityId>,
    adopt: PrimitivePointAdoption,
) -> Result<SketchEntityId, SketchPrimitiveCreationError> {
    match point {
        SketchPointInput::Owned(id) => Ok(id),
        SketchPointInput::New(position) => {
            let id = reserve_created(reservations, created)?;
            adopt(geometry, id, position).map_err(SketchPrimitiveCreationError::Sketch)?;
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

pub(super) fn append_defining_positions(
    geometry: &mut Sketch2D,
    positions: &[[f64; 2]],
    reservations: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchEntityId>, SketchPrimitiveCreationError> {
    let mut created = Vec::new();
    for position in positions {
        append_point(
            geometry,
            SketchPointInput::New(*position),
            reservations,
            &mut created,
            Sketch2D::add_defining_point,
        )?;
    }
    Ok(created)
}
