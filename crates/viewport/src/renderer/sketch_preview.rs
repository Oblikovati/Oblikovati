// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_structure::{self, SketchStructure};
use crate::ViewportSketchPreview;
use oblikovati_martensite::planar_curve::sketch::{
    SketchSamplingOptionsV1, sample_placement_preview_with_policy,
};
use rustycad_tkernel::quantity::{Color, NamedColor};
use rustycad_tkservice::aspect::AspectTypeOfLine;
use rustycad_tkv3d::v3d::V3dViewer;
#[derive(Default)]
pub(super) struct SketchPreviewOverlay {
    pub(super) source: Option<(ViewportSketchPreview, f64)>,
    structures: Vec<SketchStructure>,
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
) -> Result<Vec<SketchStructure>, String> {
    let sampled = sample_placement_preview_with_policy(
        &source.recipe,
        SketchSamplingOptionsV1 {
            deflection,
            maximum_curve_points: 4096,
        },
        source.solve_options,
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
) -> Result<SketchStructure, String> {
    let kind = if curve.construction {
        AspectTypeOfLine::Dash
    } else {
        AspectTypeOfLine::Solid
    };
    // The pinned Go dark theme uses white for transient geometry.
    sketch_structure::curve(
        viewer,
        plane,
        &curve.points,
        Color::named(NamedColor::WHITE),
        kind,
        true,
    )
}

struct HiddenPreview<'a>(&'a [SketchStructure]);
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

use rustycad_tkservice::graphic3d::Graphic3dStructureSource;
