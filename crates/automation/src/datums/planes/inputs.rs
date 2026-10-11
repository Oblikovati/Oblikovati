// SPDX-License-Identifier: GPL-2.0-only
use super::axis_point::AxisPointPlaneConstructor;
use super::*;
use oblikovati_model::feature::work_plane::{WorkPlaneOrigin, WorkPlaneReference};
pub(in crate::datums) fn prepare(
    request: CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedDatum, Error> {
    Ok(PreparedDatum {
        definition: plane_geometry(&request, snapshot)?,
        collection: DatumCollection::Planes,
        construction: request.construction,
        visible: request.visible.as_deref().copied().unwrap_or(true),
    })
}
fn plane_geometry(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    match request.kind.as_str() {
        kind if kind == WORK_PLANE_FIXED.0 => fixed_frame(request),
        kind if kind == WORK_PLANE_OFFSET.0 => offset(request, snapshot),
        kind if kind == WORK_PLANE_NORMAL_TO_CURVE.0 => {
            axis_point::create(request, snapshot, AxisPointPlaneConstructor::NormalToAxis)
        }
        kind if kind == WORK_PLANE_LINE_AND_POINT.0 => {
            axis_point::create(request, snapshot, AxisPointPlaneConstructor::ContainsLine)
        }
        kind if kind == WORK_PLANE_THREE_POINTS.0 => three_points::create(request, snapshot),
        kind if kind == WORK_PLANE_PLANE_AND_POINT.0 => parallel::create(request, snapshot),
        _ => Err(Error(format!(
            "plane constructor {:?}: expected supported {WORK_PLANE_FIXED:?}, {WORK_PLANE_OFFSET:?}, {WORK_PLANE_PLANE_AND_POINT:?}, {WORK_PLANE_THREE_POINTS:?}, {WORK_PLANE_LINE_AND_POINT:?}, or {WORK_PLANE_NORMAL_TO_CURVE:?}",
            request.kind
        ))),
    }
}
fn fixed_frame(request: &CreateWorkPlaneArgs) -> Result<PreparedGeometry, Error> {
    let origin = input::coordinates(request.origin.as_deref(), "plane origin")?;
    let x_axis = input::coordinates(request.x_axis.as_deref(), "plane X direction")?;
    let y_axis = input::coordinates(request.y_axis.as_deref(), "plane Y direction")?;
    let frame = SketchPlane::from_directions(origin, x_axis, y_axis, Default::default())
        .map_err(api_error)?;
    Ok(PreparedGeometry::Definition(WorkGeometryDefinition::Fixed(
        frame,
    )))
}
fn offset(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    let refs = request.refs.as_deref().unwrap_or_default();
    let [reference] = refs else {
        return Err(Error(format!(
            "offset references {refs:?}: expected exactly one plane reference"
        )));
    };
    let base = plane_reference(snapshot, reference)?;
    scalars::validate_offset(snapshot, &request.offset)?;
    Ok(PreparedGeometry::Offset {
        base,
        expression: request.offset.clone(),
    })
}
pub(in crate::datums) fn plane_reference(
    snapshot: &DocumentSnapshot,
    reference: &str,
) -> Result<WorkPlaneReference, Error> {
    if let Some(origin) = origin_reference(reference) {
        return Ok(WorkPlaneReference::Origin(origin));
    }
    let id = reference::decode_reference(snapshot, reference)?;
    let definition = snapshot
        .work_geometry
        .get(id)
        .map_err(api_error)?
        .definition();
    if !DatumCollection::Planes.contains(*definition) {
        return Err(Error(format!(
            "offset base {reference:?}: expected a current plane definition"
        )));
    }
    Ok(WorkPlaneReference::Feature(id))
}
fn origin_reference(reference: &str) -> Option<WorkPlaneOrigin> {
    match reference {
        WORK_REF_XY_PLANE => Some(WorkPlaneOrigin::XY),
        WORK_REF_XZ_PLANE => Some(WorkPlaneOrigin::XZ),
        WORK_REF_YZ_PLANE => Some(WorkPlaneOrigin::YZ),
        _ => None,
    }
}
