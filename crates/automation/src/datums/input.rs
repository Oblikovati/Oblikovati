// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    work_axis::WorkAxisLine,
    work_geometry::{DatumLength, WorkAxisPoints, WorkPointReference},
};

pub(super) fn definition(
    method: &str,
    request: &[u8],
    snapshot: &DocumentSnapshot,
) -> Result<PreparedDatum, Error> {
    match method {
        METHOD_WORK_POINTS_CREATE => point(decode(method, request)?, snapshot),
        METHOD_WORK_AXES_CREATE => axis(decode(method, request)?, snapshot),
        _ => Err(Error(format!(
            "datum command {method:?}: expected {DATUM_COMMAND_METHODS:?}"
        ))),
    }
}
fn point(
    request: CreateWorkPointArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedDatum, Error> {
    match request.kind.as_str() {
        "" => position_point(request),
        kind if kind == WORK_POINT_POSITION.0 => position_point(request),
        kind if kind == WORK_POINT_ON_POINT.0 => associated_point(request, snapshot),
        _ => Err(Error(format!(
            "point constructor {:?}: expected supported {:?} or {:?}",
            request.kind, WORK_POINT_POSITION, WORK_POINT_ON_POINT
        ))),
    }
}
fn position_point(request: CreateWorkPointArgs) -> Result<PreparedDatum, Error> {
    let position = coordinates(request.at.as_deref(), "point position")?;
    Ok(PreparedDatum {
        definition: WorkGeometryDefinition::Point(position.map(DatumLength::Constant)),
        construction: request.construction,
        collection: DatumCollection::Points,
    })
}
fn axis(request: CreateWorkAxisArgs, snapshot: &DocumentSnapshot) -> Result<PreparedDatum, Error> {
    let definition = match request.kind.as_str() {
        kind if kind == WORK_AXIS_LINE.0 => line(&request)?,
        kind if kind == WORK_AXIS_TWO_POINTS.0 => two_points(request.refs.as_deref(), snapshot)?,
        _ => {
            return Err(Error(format!(
                "axis constructor {:?}: expected supported {:?} or {:?}",
                request.kind, WORK_AXIS_LINE, WORK_AXIS_TWO_POINTS
            )));
        }
    };
    Ok(PreparedDatum {
        definition,
        construction: request.construction,
        collection: DatumCollection::Axes,
    })
}
fn line(request: &CreateWorkAxisArgs) -> Result<WorkGeometryDefinition, Error> {
    let origin = coordinates(request.origin.as_deref(), "axis origin")?;
    let direction = coordinates(request.direction.as_deref(), "axis direction")?;
    WorkAxisLine::new(origin, direction)
        .map(WorkGeometryDefinition::Axis)
        .map_err(api_error)
}
fn coordinates(values: Option<&[f64]>, label: &str) -> Result<[f64; 3], Error> {
    let values = values.unwrap_or_default();
    let components = <[f64; 3]>::try_from(values).map_err(|_| coordinate_error(label, values))?;
    if components.iter().any(|value| !value.is_finite()) {
        return Err(coordinate_error(label, values));
    }
    Ok(components)
}
fn coordinate_error(label: &str, values: &[f64]) -> Error {
    Error(format!(
        "{label} {values:?}: expected exactly three finite components in document model units (direction is dimensionless)"
    ))
}
fn two_points(
    refs: Option<&[String]>,
    snapshot: &DocumentSnapshot,
) -> Result<WorkGeometryDefinition, Error> {
    let refs = refs.unwrap_or_default();
    let [first, second] = refs else {
        return Err(Error(format!(
            "two-point axis references {refs:?}: expected exactly two current point references"
        )));
    };
    Ok(WorkGeometryDefinition::AxisThroughPoints(WorkAxisPoints {
        first: point_reference(snapshot, first)?,
        second: point_reference(snapshot, second)?,
    }))
}
fn point_reference(
    snapshot: &DocumentSnapshot,
    reference: &str,
) -> Result<WorkPointReference, Error> {
    if reference == WORK_REF_CENTER {
        return Ok(WorkPointReference::Origin);
    }
    let id = reference::decode_reference(snapshot, reference)?;
    let feature = snapshot.work_geometry.get(id).map_err(api_error)?;
    if !DatumCollection::Points.contains(*feature.definition()) {
        return Err(Error(format!(
            "datum {reference:?} definition {:?}: expected a current point definition",
            feature.definition()
        )));
    }
    Ok(WorkPointReference::Feature(id))
}

#[cfg(test)]
mod tests;

fn associated_point(
    request: CreateWorkPointArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedDatum, Error> {
    let refs = request.refs.as_deref().unwrap_or_default();
    let [reference] = refs else {
        return Err(Error(format!(
            "OnPoint references {refs:?}: expected exactly one current point reference"
        )));
    };
    Ok(PreparedDatum {
        definition: WorkGeometryDefinition::PointAtReference(point_reference(snapshot, reference)?),
        construction: request.construction,
        collection: DatumCollection::Points,
    })
}
