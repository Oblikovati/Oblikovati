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
    let mut batches: Vec<([f32; 4], bool, Vec<&ViewportTriangle>)> = Vec::new();
    let mut index_by_key: std::collections::HashMap<([u32; 4], bool), usize> =
        std::collections::HashMap::new();
    for (index, triangle) in triangles.iter().enumerate() {
        let color_index = match debug.colors {
            Some(true) => Some(index),
            Some(false) => triangle.face_index,
            None => None,
        };
        let color = color_index.map_or(triangle.color, mesh_palette::index_color);
        let normal_debug = debug.normals && color_index.is_none();
        let key = (color.map(f32::to_bits), normal_debug);
        if let Some(&batch_idx) = index_by_key.get(&key) {
            batches[batch_idx].2.push(triangle);
        } else {
            index_by_key.insert(key, batches.len());
            batches.push((color, normal_debug, vec![triangle]));
        }
    }
    batches
        .into_iter()
        .map(|(color, normal_debug, group)| build_batch(viewer, &group, color, normal_debug))
        .collect()
}
fn build_batch(
    viewer: &V3dViewer,
    triangles: &[&ViewportTriangle],
    color: [f32; 4],
    normal_debug: bool,
) -> Result<Handle<Graphic3dStructure>, String> {
    let structure = Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
        .map_err(|error| error.to_string())?;
    let group = structure.new_group().map_err(|error| error.to_string())?;
    let aspects = triangle_aspect(color, normal_debug)?;
    group.set_group_primitives_aspect(&aspects.cast());
    group.add_primitive_array(&batch_vertices(triangles)?.cast(), true);
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
fn batch_vertices(
    triangles: &[&ViewportTriangle],
) -> Result<Handle<Graphic3dArrayOfPrimitives>, String> {
    let count = i32::try_from(triangles.len().saturating_mul(3)).map_err(|_| {
        format!(
            "viewport batch triangles {}: expected 32-bit vertex count",
            triangles.len()
        )
    })?;
    let vertices = Handle::new(
        Graphic3dArrayOfPrimitives::new(
            Graphic3dTypeOfPrimitiveArray::Triangles,
            count,
            0,
            count,
            GRAPHIC3D_ARRAY_FLAGS_VERTEX_NORMAL,
        )
        .map_err(|error| error.to_string())?,
    );
    let mut next = 1i32;
    for triangle in triangles {
        let normals = triangle
            .normals
            .unwrap_or_else(|| [crate::render_normal::triangle_normal(triangle.positions); 3]);
        for (point, normal) in triangle.positions.into_iter().zip(normals) {
            vertices
                .add_vertex_normal(point, normal)
                .map_err(|error| error.to_string())?;
        }
        vertices
            .add_edges3(next, next + 1, next + 2)
            .map_err(|error| error.to_string())?;
        next += 3;
    }
    Ok(vertices)
}
