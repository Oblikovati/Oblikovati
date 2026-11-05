// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_vertices;
use martensite_tkernel::{
    quantity::{Color, ColorRgba, ColorSpace},
    standard::Handle,
};
use martensite_tkservice::{aspect::AspectTypeOfLine, graphic3d::*};
use martensite_tkv3d::v3d::V3dViewer;
use oblikovati_model::sketch::SketchPlane;
pub(super) struct SketchStructure(pub(super) Handle<Graphic3dStructure>);
impl Drop for SketchStructure {
    fn drop(&mut self) {
        self.0.remove();
    }
}
fn grouped_structure(
    viewer: &V3dViewer,
    transient: bool,
) -> Result<(SketchStructure, Handle<dyn Graphic3dGroup>), String> {
    let structure = SketchStructure(
        Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
            .map_err(|cause| cause.to_string())?,
    );
    // Transient geometry must not change Fit/Home or take part in model depth tests.
    // Committed geometry participates in framing; both paths share depth-independent drawing.
    structure.0.set_infinite_state(transient);
    structure.0.set_z_layer(GRAPHIC3D_ZLAYER_ID_TOPMOST);
    let group = structure.0.new_group().map_err(|cause| cause.to_string())?;
    Ok((structure, group))
}
pub(super) fn curve(
    viewer: &V3dViewer,
    plane: SketchPlane,
    points: &[[f64; 2]],
    color: &oblikovati_model::theme::Rgba,
    kind: AspectTypeOfLine,
    transient: bool,
) -> Result<SketchStructure, String> {
    let (structure, group) = grouped_structure(viewer, transient)?;
    let rgba = srgb_rgba(color)?;
    let style = Handle::new(
        Graphic3dAspectLine3d::new(*rgba.rgb(), kind, 1.).map_err(|cause| cause.to_string())?,
    );
    style.set_interior_color_rgba(rgba);
    // v0.7 line rasterization admits blending through the explicit MaskBlend aspect.
    style.set_alpha_mode(Graphic3dAlphaMode::MaskBlend, 0.);
    group.set_group_primitives_aspect(&style.cast());
    group.add_primitive_array(&sketch_vertices::polyline(plane, points)?.cast(), true);
    Ok(structure)
}
pub(super) fn point(
    viewer: &V3dViewer,
    plane: SketchPlane,
    point: [f64; 2],
    color: &oblikovati_model::theme::Rgba,
    transient: bool,
) -> Result<SketchStructure, String> {
    let (structure, group) = grouped_structure(viewer, transient)?;
    let rgba = srgb_rgba(color)?;
    let style = Handle::new(super::sketch_marker::theme_marker(rgba)?);
    style.set_interior_color_rgba(rgba);
    // v0.7 line rasterization admits blending through the explicit MaskBlend aspect.
    style.set_alpha_mode(Graphic3dAlphaMode::MaskBlend, 0.);
    group.set_group_primitives_aspect(&style.cast());
    let vertices = sketch_vertices::point(plane, point)?;
    group.add_primitive_array(&vertices.cast(), true);
    Ok(structure)
}

fn srgb_rgba(value: &oblikovati_model::theme::Rgba) -> Result<ColorRgba, String> {
    let color = Color::new(
        f64::from(value.r),
        f64::from(value.g),
        f64::from(value.b),
        ColorSpace::Srgb,
    )
    .map_err(|cause| {
        format!("sketch colour {value:?}: expected sRGB channels in [0, 1]: {cause}")
    })?;
    Ok(ColorRgba::from_rgb(color, value.a))
}
