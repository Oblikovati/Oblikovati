// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_vertices;
use oblikovati_model::sketch::SketchPlane;
use rustycad_tkernel::{quantity::Color, standard::Handle};
use rustycad_tkservice::{
    aspect::{AspectTypeOfLine, AspectTypeOfMarker},
    graphic3d::*,
};
use rustycad_tkv3d::v3d::V3dViewer;
pub(super) struct SketchStructure(pub(super) Handle<Graphic3dStructure>);
impl Drop for SketchStructure {
    fn drop(&mut self) {
        self.0.remove();
    }
}
fn structure(viewer: &V3dViewer, transient: bool) -> Result<SketchStructure, String> {
    let structure = SketchStructure(
        Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
            .map_err(|cause| cause.to_string())?,
    );
    // Transient curves must not change Fit/Home or take part in model depth tests.
    // Committed geometry participates in framing; both paths share depth-independent drawing.
    structure.0.set_infinite_state(transient);
    structure.0.set_z_layer(GRAPHIC3D_ZLAYER_ID_TOPMOST);
    structure.0.new_group().map_err(|cause| cause.to_string())?;
    Ok(structure)
}
pub(super) fn curve(
    viewer: &V3dViewer,
    plane: SketchPlane,
    points: &[[f64; 2]],
    color: Color,
    kind: AspectTypeOfLine,
    transient: bool,
) -> Result<SketchStructure, String> {
    let structure = structure(viewer, transient)?;
    let group = structure
        .0
        .current_group()
        .map_err(|cause| cause.to_string())?;
    let style = Handle::new(
        Graphic3dAspectLine3d::new(color, kind, 1.).map_err(|cause| cause.to_string())?,
    );
    group.set_group_primitives_aspect(&style.cast());
    group.add_primitive_array(&sketch_vertices::polyline(plane, points)?.cast(), true);
    Ok(structure)
}
pub(super) fn point(
    viewer: &V3dViewer,
    plane: SketchPlane,
    point: [f64; 2],
    color: Color,
) -> Result<SketchStructure, String> {
    let structure = structure(viewer, false)?;
    let group = structure
        .0
        .current_group()
        .map_err(|cause| cause.to_string())?;
    let style = Handle::new(
        Graphic3dAspectMarker3d::new(AspectTypeOfMarker::OPlus, color, 1.)
            .map_err(|cause| cause.to_string())?,
    );
    group.set_group_primitives_aspect(&style.cast());
    let vertices = sketch_vertices::point(plane, point)?;
    group.add_primitive_array(&vertices.cast(), true);
    Ok(structure)
}
