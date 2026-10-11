// SPDX-License-Identifier: GPL-2.0-only
use super::{DatumLength, DatumLengthError, WorkGeometryError};
use crate::{feature::FeatureId, parameter::ParameterCollection};
use std::{error::Error, fmt};

/// A current datum point identified independently of collection order.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum WorkPointReference {
    Origin,
    Feature(FeatureId),
}
/// Ordered stable point references determine the directed axis; no evaluated positions are stored.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkAxisPoints {
    pub first: WorkPointReference,
    pub second: WorkPointReference,
}
/// A coordinate decline names its component and retains the shared scalar cause.
#[derive(Clone, Debug, PartialEq)]
pub struct WorkPointCoordinateError {
    pub component: usize,
    pub cause: DatumLengthError,
}
impl WorkPointReference {
    /// Resolve the current point or the document origin; example: `reference.position(&evaluation)?`.
    pub fn position(
        self,
        evaluation: &impl crate::feature::work_plane::WorkPlaneResolver,
    ) -> Result<[f64; 3], WorkGeometryError> {
        match self {
            Self::Origin => Ok([0.0; 3]),
            Self::Feature(id) => evaluation.point(id),
        }
    }
}
pub(super) fn resolve_position(
    coordinates: [DatumLength; 3],
    parameters: &ParameterCollection<u64>,
) -> Result<[f64; 3], WorkGeometryError> {
    Ok([
        resolve_coordinate(0, coordinates[0], parameters)?,
        resolve_coordinate(1, coordinates[1], parameters)?,
        resolve_coordinate(2, coordinates[2], parameters)?,
    ])
}
fn resolve_coordinate(
    component: usize,
    coordinate: DatumLength,
    parameters: &ParameterCollection<u64>,
) -> Result<f64, WorkGeometryError> {
    coordinate.resolve(parameters).map_err(|cause| {
        WorkGeometryError::Coordinate(WorkPointCoordinateError { component, cause })
    })
}
pub(crate) fn validate_position(coordinates: [DatumLength; 3]) -> Result<(), WorkGeometryError> {
    for (component, coordinate) in coordinates.into_iter().enumerate() {
        coordinate.validate_source().map_err(|cause| {
            WorkGeometryError::Coordinate(WorkPointCoordinateError { component, cause })
        })?;
    }
    Ok(())
}
impl fmt::Display for WorkPointCoordinateError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            output,
            "datum point component {}: {}",
            self.component, self.cause
        )
    }
}
impl Error for WorkPointCoordinateError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(&self.cause)
    }
}
