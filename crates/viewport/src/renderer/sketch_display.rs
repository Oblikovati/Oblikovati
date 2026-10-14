// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_structure::{self, SketchStructure};
use crate::{ViewportSketchDecline, ViewportSketchDisplay};
use oblikovati_martensite::planar_curve::sketch::{
    SketchDisplayEntity, SketchDisplayPrimitive, SketchSamplingOptionsV1, sample_sketch_display,
};
use rustycad_tkernel::quantity::{Color, ColorSpace};
use rustycad_tkservice::aspect::AspectTypeOfLine;
use rustycad_tkv3d::v3d::V3dViewer;
#[derive(Default)]
pub(super) struct SketchDisplayOverlay {
    pub(super) source: Option<(Vec<ViewportSketchDisplay>, f64)>,
    pub(super) declines: Vec<ViewportSketchDecline>,
    structures: Vec<SketchStructure>,
}
impl SketchDisplayOverlay {
    pub(super) fn is_empty(&self) -> bool {
        self.structures.is_empty()
    }

    pub(super) fn submit(
        &mut self,
        viewer: &V3dViewer,
        source: &[ViewportSketchDisplay],
        deflection: f64,
    ) -> Result<bool, String> {
        if self
            .source
            .as_ref()
            .is_some_and(|(previous, error)| previous == source && *error == deflection)
        {
            return Ok(false);
        }
        let replacement = Self::replacement(viewer, source, deflection)?;
        for structure in &replacement.structures {
            structure.0.display();
        }
        *self = replacement;
        Ok(true)
    }
    fn replacement(
        viewer: &V3dViewer,
        source: &[ViewportSketchDisplay],
        deflection: f64,
    ) -> Result<Self, String> {
        let mut result = Self {
            source: Some((source.to_vec(), deflection)),
            ..Self::default()
        };
        for sketch in source {
            result.append_sketch(viewer, sketch, deflection)?;
        }
        Ok(result)
    }
    fn append_sketch(
        &mut self,
        viewer: &V3dViewer,
        sketch: &ViewportSketchDisplay,
        deflection: f64,
    ) -> Result<(), String> {
        let report = sample_sketch_display(
            &sketch.geometry,
            SketchSamplingOptionsV1 {
                deflection,
                maximum_curve_points: 4096,
            },
        )?;
        self.declines.extend(
            report
                .declines
                .into_iter()
                .map(|decline| display_decline(sketch, decline.entity, decline.cause)),
        );
        for point in report.center_markers {
            self.append_marker(viewer, sketch, point);
        }
        for entity in report.entities {
            self.append_entity(viewer, sketch, &entity);
        }
        Ok(())
    }
    fn append_marker(
        &mut self,
        viewer: &V3dViewer,
        sketch: &ViewportSketchDisplay,
        point: oblikovati_model::sketch::SketchPoint2D,
    ) {
        let entity = SketchDisplayEntity {
            entity: point.id,
            construction: false,
            centerline: false,
            primitive: SketchDisplayPrimitive::Point(point.position),
        };
        self.append_entity(viewer, sketch, &entity);
    }
    fn append_entity(
        &mut self,
        viewer: &V3dViewer,
        sketch: &ViewportSketchDisplay,
        entity: &SketchDisplayEntity,
    ) {
        match build_entity(viewer, sketch, entity) {
            Ok(structure) => self.structures.push(structure),
            Err(cause) => self
                .declines
                .push(display_decline(sketch, entity.entity, cause)),
        }
    }
    pub(super) fn clear(&mut self) {
        self.structures.clear();
        self.declines.clear();
        self.source = None;
    }
}
fn build_entity(
    viewer: &V3dViewer,
    sketch: &ViewportSketchDisplay,
    entity: &SketchDisplayEntity,
) -> Result<SketchStructure, String> {
    // Go's default dark theme maps TokenSketchGeometry to wire_edit #a0a0a0.
    let color = Color::new(160. / 255., 160. / 255., 160. / 255., ColorSpace::Srgb)
        .map_err(|cause| cause.to_string())?;
    let kind = entity_line_style(entity);
    match &entity.primitive {
        SketchDisplayPrimitive::Point(point) => {
            sketch_structure::point(viewer, sketch.plane, *point, color)
        }
        SketchDisplayPrimitive::Curve(points) => {
            sketch_structure::curve(viewer, sketch.plane, points, color, kind, false)
        }
    }
}
impl super::RenderView {
    pub(crate) fn sketch_display(
        &mut self,
        source: &[ViewportSketchDisplay],
    ) -> Result<bool, String> {
        let deflection = self.sketch_deflection()?;
        self.sketches.submit(&self.viewer, source, deflection)
    }
    pub(crate) fn refresh_sketch_display(&mut self) -> Result<bool, String> {
        let deflection = self.sketch_deflection()?;
        let Some((source, previous)) = self.sketches.source.as_ref() else {
            return Ok(false);
        };
        if *previous == deflection {
            return Ok(false);
        }
        let source = source.clone();
        self.sketches.submit(&self.viewer, &source, deflection)
    }
    pub(crate) fn sketch_declines(&self) -> &[ViewportSketchDecline] {
        &self.sketches.declines
    }
}

use rustycad_tkservice::graphic3d::Graphic3dStructureSource;

fn entity_line_style(entity: &SketchDisplayEntity) -> AspectTypeOfLine {
    if entity.centerline {
        return AspectTypeOfLine::Dotdash;
    }
    if entity.construction {
        return AspectTypeOfLine::Dash;
    }
    AspectTypeOfLine::Solid
}

fn display_decline(
    sketch: &ViewportSketchDisplay,
    entity: oblikovati_model::sketch::SketchEntityId,
    cause: String,
) -> ViewportSketchDecline {
    ViewportSketchDecline {
        sketch: sketch.sketch,
        entity,
        cause,
    }
}
