// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(in crate::datums) fn prepare(request: CreateWorkPlaneArgs) -> Result<PreparedDatum, Error> {
    if request.kind != WORK_PLANE_FIXED.0 {
        return Err(Error(format!(
            "plane constructor {:?}: expected supported {:?}; associative expression and other constructors require their owned definitions",
            request.kind, WORK_PLANE_FIXED
        )));
    }
    let origin = input::coordinates(request.origin.as_deref(), "plane origin")?;
    let x_axis = input::coordinates(request.x_axis.as_deref(), "plane X direction")?;
    let y_axis = input::coordinates(request.y_axis.as_deref(), "plane Y direction")?;
    let frame = SketchPlane::from_directions(origin, x_axis, y_axis, Default::default())
        .map_err(api_error)?;
    Ok(PreparedDatum {
        definition: WorkGeometryDefinition::Fixed(frame),
        collection: DatumCollection::Planes,
        construction: request.construction,
        visible: request.visible.as_deref().copied().unwrap_or(true),
    })
}
