// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkernel::standard::Handle;
use martensite_tkservice::graphic3d::*;
use oblikovati_model::sketch::SketchPlane;
pub(super) fn polylines<P: AsRef<[[f64; 2]]>>(
    plane: SketchPlane,
    paths: &[P],
) -> Result<Handle<Graphic3dArrayOfPrimitives>, String> {
    let bounds = i32::try_from(paths.len())
        .map_err(|_| format!("sketch paths {}: expected an i32 count", paths.len()))?;
    let count = paths
        .iter()
        .try_fold(0i32, |total, path| {
            let count = i32::try_from(path.as_ref().len()).ok()?;
            total.checked_add(count)
        })
        .ok_or("sketch paths exceed the total i32 vertex limit")?;
    let vertices = Handle::new(
        Graphic3dArrayOfPrimitives::new(
            Graphic3dTypeOfPrimitiveArray::Polylines,
            count,
            bounds,
            0,
            0,
        )
        .map_err(|cause| cause.to_string())?,
    );
    for path in paths {
        append_path(&vertices, plane, path.as_ref())?;
    }
    Ok(vertices)
}

fn append_path(
    vertices: &Graphic3dArrayOfPrimitives,
    plane: SketchPlane,
    points: &[[f64; 2]],
) -> Result<(), String> {
    vertices
        .add_bound(points.len() as i32)
        .map_err(|cause| cause.to_string())?;
    for &point in points {
        let [x, y, z] = render_position(plane, point)?;
        vertices
            .add_vertex_f64(x, y, z)
            .map_err(|cause| cause.to_string())?;
    }
    Ok(())
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
