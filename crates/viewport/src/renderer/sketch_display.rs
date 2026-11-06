// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_structure::{self, SketchStructure};
use crate::{ViewportSketchDecline, ViewportSketchDisplay};
use martensite_tkservice::aspect::AspectTypeOfLine;
use martensite_tkv3d::v3d::V3dViewer;
use oblikovati_martensite::planar_curve::sketch::{
    SketchDisplayEntity, SketchDisplayPrimitive, SketchSamplingOptionsV1, sample_sketch_display,
};
use oblikovati_model::theme::Rgba;
#[derive(Default)]
pub(super) struct SketchDisplayOverlay {
    pub(super) source: Option<(Vec<ViewportSketchDisplay>, f64)>,
    pub(super) declines: Vec<ViewportSketchDecline>,
    structures: Vec<SketchStructure>,
    color: Rgba,
    selected_color: Rgba,
    selected: crate::ViewportSketchSelection,
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
        palette: &crate::ViewportSketchPalette,
        selected: &crate::ViewportSketchSelection,
    ) -> Result<bool, String> {
        if self.source.as_ref().is_some_and(|(previous, error)| {
            previous == source
                && *error == deflection
                && self.color == palette.geometry
                && self.selected_color == palette.selected
                && self.selected == *selected
        }) {
            return Ok(false);
        }
        let replacement = Self::replacement(viewer, source, deflection, palette, selected)?;
        replacement.show();
        *self = replacement;
        Ok(true)
    }
    pub(super) fn replacement(
        viewer: &V3dViewer,
        source: &[ViewportSketchDisplay],
        deflection: f64,
        palette: &crate::ViewportSketchPalette,
        selected: &crate::ViewportSketchSelection,
    ) -> Result<Self, String> {
        let mut result = Self {
            source: Some((source.to_vec(), deflection)),
            color: palette.geometry.clone(),
            selected_color: palette.selected.clone(),
            selected: selected.clone(),
            ..Self::default()
        };
        for sketch in source {
            result.append_sketch(viewer, sketch, deflection)?;
        }
        Ok(result)
    }
    pub(super) fn show(&self) {
        for structure in &self.structures {
            structure.0.display();
        }
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
        let marked = report
            .center_markers
            .iter()
            .map(|point| point.id)
            .chain(report.entities.iter().map(|entity| entity.entity))
            .collect::<std::collections::BTreeSet<_>>();
        for point in sketch
            .geometry
            .points()
            .iter()
            .filter(|point| {
                !marked.contains(&point.id)
                    && self.selected.entities.contains(&(sketch.sketch, point.id))
            })
            .copied()
            .collect::<Vec<_>>()
        {
            self.append_marker(viewer, sketch, point);
        }
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
        let color = if self.selected.sketches.contains(&sketch.sketch)
            || self
                .selected
                .entities
                .contains(&(sketch.sketch, entity.entity))
        {
            &self.selected_color
        } else {
            &self.color
        };
        match build_entity(viewer, sketch, entity, color) {
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
    color: &Rgba,
) -> Result<SketchStructure, String> {
    let kind = entity_line_style(entity);
    match &entity.primitive {
        SketchDisplayPrimitive::Point(point) => {
            sketch_structure::point(viewer, sketch.plane, *point, color, false)
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
        self.sketches.submit(
            &self.viewer,
            source,
            deflection,
            &self.sketch_palette,
            &self.sketch_selection,
        )
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
        self.sketches.submit(
            &self.viewer,
            &source,
            deflection,
            &self.sketch_palette,
            &self.sketch_selection,
        )
    }
    pub(crate) fn sketch_declines(&self) -> &[ViewportSketchDecline] {
        &self.sketches.declines
    }
}

use martensite_tkservice::graphic3d::Graphic3dStructureSource;

fn entity_line_style(entity: &SketchDisplayEntity) -> AspectTypeOfLine {
    curve_role_line_style(entity.construction, entity.centerline)
}
pub(super) fn curve_role_line_style(construction: bool, centerline: bool) -> AspectTypeOfLine {
    if centerline {
        return AspectTypeOfLine::Dotdash;
    }
    if construction {
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

#[cfg(test)]
mod role_style_tests {
    use super::*;
    #[test]
    fn axis_style_has_priority_over_its_effective_construction_role() {
        assert_eq!(curve_role_line_style(false, false), AspectTypeOfLine::Solid);
        assert_eq!(curve_role_line_style(true, false), AspectTypeOfLine::Dash);
        assert_eq!(
            curve_role_line_style(false, true),
            AspectTypeOfLine::Dotdash
        );
        assert_eq!(curve_role_line_style(true, true), AspectTypeOfLine::Dotdash);
    }
}
