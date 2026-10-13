// SPDX-License-Identifier: GPL-2.0-only
use crate::ViewportSketchPreview;
use oblikovati_martensite::planar_curve::sketch::{
    PlacementSamplingOptionsV1, sample_placement_preview,
};
use rustycad_tkernel::{
    quantity::{Color, NamedColor},
    standard::Handle,
};
use rustycad_tkservice::{aspect::AspectTypeOfLine, graphic3d::*};
use rustycad_tkv3d::v3d::V3dViewer;
mod vertices;
#[derive(Default)]
pub(super) struct SketchPreviewOverlay {
    pub(super) source: Option<(ViewportSketchPreview, f64)>,
    structures: Vec<PreviewStructure>,
}
struct PreviewStructure(Handle<Graphic3dStructure>);
impl Drop for PreviewStructure {
    fn drop(&mut self) {
        self.0.remove();
    }
}
impl SketchPreviewOverlay {
    pub(super) fn submit(
        &mut self,
        viewer: &V3dViewer,
        source: Option<&ViewportSketchPreview>,
        deflection: f64,
    ) -> Result<bool, String> {
        let key = source.map(|source| (source.clone(), deflection));
        if self.source == key {
            return Ok(false);
        }
        let replacement = source
            .map(|source| build(viewer, source, deflection))
            .transpose()?
            .unwrap_or_default();
        for structure in &replacement {
            structure.0.display();
        }
        self.structures = replacement;
        self.source = key;
        Ok(true)
    }
    pub(super) fn clear(&mut self) {
        self.structures.clear();
        self.source = None;
    }
}
fn build(
    viewer: &V3dViewer,
    source: &ViewportSketchPreview,
    deflection: f64,
) -> Result<Vec<PreviewStructure>, String> {
    let sampled = sample_placement_preview(
        &source.recipe,
        PlacementSamplingOptionsV1 {
            deflection,
            maximum_curve_points: 4096,
        },
    )?;
    sampled
        .iter()
        .map(|curve| build_member(viewer, source.plane, curve))
        .collect()
}
fn build_member(
    viewer: &V3dViewer,
    plane: oblikovati_model::sketch::SketchPlane,
    curve: &oblikovati_martensite::planar_curve::sketch::PlacementPreviewCurve,
) -> Result<PreviewStructure, String> {
    let structure = new_structure(viewer, curve.construction)?;
    let group = structure
        .0
        .current_group()
        .map_err(|cause| cause.to_string())?;
    let vertices = vertices::polyline(plane, &curve.points)?;
    group.add_primitive_array(&vertices.cast(), true);
    Ok(structure)
}
fn new_structure(viewer: &V3dViewer, construction: bool) -> Result<PreviewStructure, String> {
    let structure = PreviewStructure(
        Graphic3dStructure::new(&viewer.structure_manager(), &Handle::default())
            .map_err(|cause| cause.to_string())?,
    );
    // Transient curves must not change Fit/Home or take part in model depth tests.
    structure.0.set_infinite_state(true);
    structure.0.set_z_layer(GRAPHIC3D_ZLAYER_ID_TOPMOST);
    let group = structure.0.new_group().map_err(|cause| cause.to_string())?;
    let style = line_style(construction)?;
    group.set_group_primitives_aspect(&style.cast());
    Ok(structure)
}

fn line_style(construction: bool) -> Result<Handle<Graphic3dAspectLine3d>, String> {
    let kind = if construction {
        AspectTypeOfLine::Dash
    } else {
        AspectTypeOfLine::Solid
    };
    // The pinned Go dark theme uses white for transient geometry.
    Graphic3dAspectLine3d::new(Color::named(NamedColor::WHITE), kind, 1.)
        .map(Handle::new)
        .map_err(|cause| cause.to_string())
}

struct HiddenPreview<'a>(&'a [PreviewStructure]);
impl Drop for HiddenPreview<'_> {
    fn drop(&mut self) {
        for structure in self.0 {
            structure.0.display();
        }
    }
}
impl SketchPreviewOverlay {
    pub(super) fn without_preview<T>(&self, operation: impl FnOnce() -> T) -> T {
        // v0.2.5 ZFitAll includes infinite structures in its graphic depth bounds.
        // Exclude transient presentation for the whole fit, including error returns.
        let _restore = HiddenPreview(&self.structures);
        for structure in &self.structures {
            structure.0.erase();
        }
        operation()
    }
}
