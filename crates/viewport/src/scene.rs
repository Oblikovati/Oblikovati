// SPDX-License-Identifier: GPL-2.0-only
use crate::ViewportTriangle;
use martensite_tkernel::{
    quantity::{Color, ColorRgba, NamedColor},
    standard::Handle,
};
use martensite_tkservice::graphic3d::*;
use martensite_tkv3d::v3d::V3dViewer;
#[derive(Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct MeshDebug {
    pub normals: bool,
    /// None for ordinary shading, Some(false) for faces, Some(true) for triangles.
    pub colors: Option<bool>,
}

mod mesh_palette;

pub(crate) fn build_scene(
    viewer: &V3dViewer,
    triangles: &[ViewportTriangle],
    debug: MeshDebug,
) -> Result<Vec<Handle<Graphic3dStructure>>, String> {
    let mut structures = Vec::with_capacity(triangles.len());
    for (index, triangle) in triangles.iter().enumerate() {
        let color_index = match debug.colors {
            Some(true) => Some(index),
            Some(false) => triangle.face_index,
            None => None,
        };
        let color = color_index.map_or(triangle.color, mesh_palette::index_color);
        let normals = debug.normals && color_index.is_none();
        structures.push(build_triangle(viewer, triangle, color, normals)?);
    }
    Ok(structures)
}
fn build_triangle(
    viewer: &V3dViewer,
    triangle: &ViewportTriangle,
    color: [f32; 4],
    normal_debug: bool,
) -> Result<Handle<Graphic3dStructure>, String> {
    let structure = Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
        .map_err(|error| error.to_string())?;
    let group = structure.new_group().map_err(|error| error.to_string())?;
    let aspects = triangle_aspect(color, normal_debug)?;
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
