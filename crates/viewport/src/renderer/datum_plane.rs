// SPDX-License-Identifier: GPL-2.0-only
//! Depth-tested datum overlays, separate from solid scene structures.
use super::{RenderView, sketch_structure::SketchStructure};
use crate::{ViewportDatumAxis, ViewportDatumPlane};
use martensite_tkernel::{
    quantity::{Color, ColorRgba, ColorSpace},
    standard::Handle,
};
use martensite_tkservice::{aspect::AspectTypeOfLine, graphic3d::*};
use martensite_tkv3d::v3d::V3dViewer;
use oblikovati_model::theme::Rgba;

#[derive(Default)]
pub(super) struct DatumOverlay {
    source: Vec<ViewportDatumPlane>,
    axes_source: Vec<ViewportDatumAxis>,
    structures: Vec<SketchStructure>,
    axis_structures: Vec<SketchStructure>,
}

impl RenderView {
    pub(crate) fn datum_planes(&mut self, planes: &[ViewportDatumPlane]) -> Result<bool, String> {
        if self.datums.source == planes {
            return Ok(false);
        }
        let structures = build_planes(&self.viewer, planes)?;
        for structure in &structures {
            structure.0.display();
        }
        self.datums.source = planes.to_vec();
        self.datums.structures = structures;
        Ok(true)
    }

    pub(crate) fn datum_axes(&mut self, axes: &[ViewportDatumAxis]) -> Result<bool, String> {
        if self.datums.axes_source == axes {
            return Ok(false);
        }
        let structures = build_axes(&self.viewer, axes)?;
        for structure in &structures {
            structure.0.display();
        }
        self.datums.axes_source = axes.to_vec();
        self.datums.axis_structures = structures;
        Ok(true)
    }
}

impl DatumOverlay {
    pub(super) fn clear(&mut self) {
        self.structures.clear();
        self.axis_structures.clear();
        self.source.clear();
        self.axes_source.clear();
    }
}

fn build_planes(
    viewer: &V3dViewer,
    planes: &[ViewportDatumPlane],
) -> Result<Vec<SketchStructure>, String> {
    let mut structures = Vec::with_capacity(planes.len() * 2);
    for plane in planes {
        plane.validate()?;
        structures.push(fill(viewer, plane)?);
        structures.push(border(viewer, plane)?);
    }
    Ok(structures)
}

fn build_axes(
    viewer: &V3dViewer,
    axes: &[ViewportDatumAxis],
) -> Result<Vec<SketchStructure>, String> {
    let mut structures = Vec::with_capacity(axes.len());
    for axis in axes {
        axis.validate()?;
        structures.push(axis_line(viewer, axis)?);
    }
    Ok(structures)
}

fn axis_line(viewer: &V3dViewer, axis: &ViewportDatumAxis) -> Result<SketchStructure, String> {
    let (structure, group) = new_group(viewer)?;
    let rgba = color(&axis.color)?;
    let line_type = if axis.selected {
        AspectTypeOfLine::Solid
    } else {
        AspectTypeOfLine::Dotdash
    };
    let style = Handle::new(
        Graphic3dAspectLine3d::new(*rgba.rgb(), line_type, f64::from(axis.width))
            .map_err(|cause| cause.to_string())?,
    );
    style.set_interior_color_rgba(rgba);
    style.set_alpha_mode(Graphic3dAlphaMode::MaskBlend, 0.);
    group.set_group_primitives_aspect(&style.cast());
    let vertices = Handle::new(
        Graphic3dArrayOfPrimitives::new(Graphic3dTypeOfPrimitiveArray::Polylines, 2, 1, 0, 0)
            .map_err(|cause| cause.to_string())?,
    );
    vertices.add_bound(2).map_err(|cause| cause.to_string())?;
    for point in axis.endpoints {
        vertices
            .add_vertex_f64(point[0], point[1], point[2])
            .map_err(|cause| cause.to_string())?;
    }
    group.add_primitive_array(&vertices.cast(), true);
    Ok(structure)
}

fn new_group(viewer: &V3dViewer) -> Result<(SketchStructure, Handle<dyn Graphic3dGroup>), String> {
    let structure = SketchStructure(
        Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
            .map_err(|cause| cause.to_string())?,
    );
    let group = structure.0.new_group().map_err(|cause| cause.to_string())?;
    Ok((structure, group))
}

fn color(value: &Rgba) -> Result<ColorRgba, String> {
    let rgb = Color::new(
        f64::from(value.r),
        f64::from(value.g),
        f64::from(value.b),
        ColorSpace::Srgb,
    )
    .map_err(|cause| format!("datum color {value:?}: expected sRGB channels: {cause}"))?;
    Ok(ColorRgba::from_rgb(rgb, value.a))
}

fn fill(viewer: &V3dViewer, plane: &ViewportDatumPlane) -> Result<SketchStructure, String> {
    let (structure, group) = new_group(viewer)?;
    let style = Handle::new(Graphic3dAspects::default());
    style.set_shading_model(Graphic3dTypeOfShadingModel::Phong);
    style.set_interior_color_rgba(color(&plane.fill)?);
    let mut material = style.front_material();
    material
        .set_transparency(1. - plane.fill.a)
        .map_err(|cause| cause.to_string())?;
    style.set_front_material(material);
    style.set_alpha_mode(Graphic3dAlphaMode::Blend, 0.);
    group.set_group_primitives_aspect(&style.cast());
    let triangles = fill_vertices(plane)?;
    group.add_primitive_array(&triangles.cast(), true);
    Ok(structure)
}

fn fill_vertices(plane: &ViewportDatumPlane) -> Result<Handle<Graphic3dArrayOfPrimitives>, String> {
    let vertices = Handle::new(
        Graphic3dArrayOfPrimitives::new(
            Graphic3dTypeOfPrimitiveArray::Triangles,
            4,
            0,
            6,
            GRAPHIC3D_ARRAY_FLAGS_VERTEX_NORMAL,
        )
        .map_err(|cause| cause.to_string())?,
    );
    let positions = plane.corners.map(|point| point.map(|value| value as f32));
    let normal = crate::render_normal::triangle_normal([positions[0], positions[1], positions[2]]);
    if normal == [0.; 3] {
        return Err(format!(
            "datum corners {:?}: expected a nondegenerate rectangle",
            plane.corners
        ));
    }
    for point in positions {
        vertices
            .add_vertex_normal(point, normal)
            .map_err(|cause| cause.to_string())?;
    }
    vertices
        .add_edges3(1, 2, 3)
        .map_err(|cause| cause.to_string())?;
    vertices
        .add_edges3(1, 3, 4)
        .map_err(|cause| cause.to_string())?;
    Ok(vertices)
}

fn border(viewer: &V3dViewer, plane: &ViewportDatumPlane) -> Result<SketchStructure, String> {
    let (structure, group) = new_group(viewer)?;
    let rgba = color(&plane.border)?;
    let style = Handle::new(
        Graphic3dAspectLine3d::new(*rgba.rgb(), AspectTypeOfLine::Solid, 1.)
            .map_err(|cause| cause.to_string())?,
    );
    style.set_interior_color_rgba(rgba);
    style.set_alpha_mode(Graphic3dAlphaMode::MaskBlend, 0.);
    group.set_group_primitives_aspect(&style.cast());
    group.add_primitive_array(&border_vertices(plane)?.cast(), true);
    Ok(structure)
}

fn border_vertices(
    plane: &ViewportDatumPlane,
) -> Result<Handle<Graphic3dArrayOfPrimitives>, String> {
    let vertices = Handle::new(
        Graphic3dArrayOfPrimitives::new(Graphic3dTypeOfPrimitiveArray::Polylines, 5, 1, 0, 0)
            .map_err(|cause| cause.to_string())?,
    );
    vertices.add_bound(5).map_err(|cause| cause.to_string())?;
    for point in plane.corners.into_iter().chain([plane.corners[0]]) {
        vertices
            .add_vertex_f64(point[0], point[1], point[2])
            .map_err(|cause| cause.to_string())?;
    }
    Ok(vertices)
}
