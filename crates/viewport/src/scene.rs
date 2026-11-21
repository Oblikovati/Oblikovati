// SPDX-License-Identifier: GPL-2.0-only
use crate::ViewportTriangle;
use martensite_tkernel::{
    quantity::{Color, ColorRgba, NamedColor},
    standard::Handle,
};
use martensite_tkservice::graphic3d::*;
use martensite_tkv3d::v3d::V3dViewer;
pub(crate) fn build_scene(
    viewer: &V3dViewer,
    triangles: &[ViewportTriangle],
) -> Result<Vec<Handle<Graphic3dStructure>>, String> {
    let mut structures = Vec::new();
    for triangle in triangles {
        structures.push(build_triangle(viewer, triangle)?);
    }
    Ok(structures)
}
fn build_triangle(
    viewer: &V3dViewer,
    triangle: &ViewportTriangle,
) -> Result<Handle<Graphic3dStructure>, String> {
    let structure = Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
        .map_err(|error| error.to_string())?;
    let group = structure.new_group().map_err(|error| error.to_string())?;
    let aspects = Handle::new(Graphic3dAspects::default());
    aspects.set_polygon_offsets(0, 0., 0.);
    aspects.set_shading_model(Graphic3dTypeOfShadingModel::Phong);
    // #4044: the default material multiplies ambient by 0.25 before the scene
    // color. Use the full scene color as the ambient reflectance, like the Go
    // shaded view; the viewer still controls the ambient light intensity.
    aspects.change_front_material(|material| {
        material.set_ambient_color(Color::named(NamedColor::WHITE))
    });
    let [red, green, blue, alpha] = triangle.color;
    aspects.set_interior_color_rgba(
        ColorRgba::new(red, green, blue, alpha).map_err(|error| error.to_string())?,
    );
    group.set_group_primitives_aspect(&aspects.cast());
    group.add_primitive_array(&triangle_vertices(triangle)?.cast(), true);
    Ok(structure)
}
fn triangle_vertices(
    triangle: &ViewportTriangle,
) -> Result<Handle<Graphic3dArrayOfPrimitives>, String> {
    let vertices = Handle::new(
        Graphic3dArrayOfPrimitives::new(
            Graphic3dTypeOfPrimitiveArray::Triangles,
            3,
            0,
            3,
            GRAPHIC3D_ARRAY_FLAGS_VERTEX_NORMAL,
        )
        .map_err(|error| error.to_string())?,
    );
    let normal = crate::render_normal::triangle_normal(triangle.positions);
    for point in triangle.positions {
        vertices
            .add_vertex_normal(point, normal)
            .map_err(|error| error.to_string())?;
    }
    vertices
        .add_edges3(1, 2, 3)
        .map_err(|error| error.to_string())?;
    Ok(vertices)
}
