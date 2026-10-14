// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::SketchPlane;
use rustycad_tkernel::standard::Handle;
use rustycad_tkservice::graphic3d::*;
pub(super) fn polyline(
    plane: SketchPlane,
    points: &[[f64; 2]],
) -> Result<Handle<Graphic3dArrayOfPrimitives>, String> {
    let count = i32::try_from(points.len())
        .map_err(|_| format!("preview vertices {}: expected an i32 count", points.len()))?;
    let vertices = Handle::new(
        Graphic3dArrayOfPrimitives::new(Graphic3dTypeOfPrimitiveArray::Polylines, count, 1, 0, 0)
            .map_err(|cause| cause.to_string())?,
    );
    vertices
        .add_bound(count)
        .map_err(|cause| cause.to_string())?;
    for point in points {
        let position = render_position(plane, *point)?;
        vertices
            .add_vertex_f64(position[0], position[1], position[2])
            .map_err(|cause| cause.to_string())?;
    }
    Ok(vertices)
}

pub(super) fn render_position(plane: SketchPlane, point: [f64; 2]) -> Result<[f64; 3], String> {
    let position = plane.to_model(point).map_err(|cause| cause.to_string())?;
    if position.iter().any(|value| !(*value as f32).is_finite()) {
        return Err(format!(
            "preview position {position:?}: expected finite f32 render coordinates"
        ));
    }
    Ok(position)
}

pub(super) fn point(
    plane: SketchPlane,
    point: [f64; 2],
) -> Result<Handle<Graphic3dArrayOfPrimitives>, String> {
    let vertices = Handle::new(
        Graphic3dArrayOfPrimitives::new(Graphic3dTypeOfPrimitiveArray::Points, 1, 0, 0, 0)
            .map_err(|cause| cause.to_string())?,
    );
    let position = render_position(plane, point)?;
    vertices
        .add_vertex_f64(position[0], position[1], position[2])
        .map_err(|cause| cause.to_string())?;
    Ok(vertices)
}
