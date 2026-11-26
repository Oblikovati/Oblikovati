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
    normal_debug: bool,
) -> Result<Vec<Handle<Graphic3dStructure>>, String> {
    let mut structures = Vec::new();
    for triangle in triangles {
        structures.push(build_triangle(viewer, triangle, normal_debug)?);
    }
    Ok(structures)
}
fn build_triangle(
    viewer: &V3dViewer,
    triangle: &ViewportTriangle,
    normal_debug: bool,
) -> Result<Handle<Graphic3dStructure>, String> {
    let structure = Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
        .map_err(|error| error.to_string())?;
    let group = structure.new_group().map_err(|error| error.to_string())?;
    let aspects = triangle_aspect(triangle.color, normal_debug)?;
    group.set_group_primitives_aspect(&aspects.cast());
    group.add_primitive_array(&triangle_vertices(triangle)?.cast(), true);
    Ok(structure)
}
fn triangle_aspect(
    color: [f32; 4],
    normal_debug: bool,
) -> Result<Handle<Graphic3dAspects>, String> {
    let aspects = Handle::new(Graphic3dAspects::default());
    aspects.set_polygon_offsets(0, 0., 0.);
    if normal_debug {
        aspects.set_shading_model(Graphic3dTypeOfShadingModel::Unlit);
        aspects.set_distinguish(true);
        aspects.set_face_culling(Graphic3dTypeOfBackfacingModel::DoubleSided);
        aspects.set_interior_color(Color::named(NamedColor::GREEN));
        aspects.set_back_interior_color(Color::named(NamedColor::RED));
        return Ok(aspects);
    }
    aspects.set_shading_model(Graphic3dTypeOfShadingModel::Phong);
    // #4044: use the full scene color as ambient reflectance, as in the Go shaded view.
    aspects.change_front_material(|material| {
        material.set_ambient_color(Color::named(NamedColor::WHITE))
    });
    let [red, green, blue, alpha] = color;
    aspects.set_interior_color_rgba(
        ColorRgba::new(red, green, blue, alpha).map_err(|error| error.to_string())?,
    );
    Ok(aspects)
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
