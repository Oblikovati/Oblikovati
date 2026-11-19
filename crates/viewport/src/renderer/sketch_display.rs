// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_structure::{self, SketchStructure};
use crate::{ViewportSketchDecline, ViewportSketchDisplay};
use martensite_tkservice::aspect::AspectTypeOfLine;
use martensite_tkv3d::v3d::V3dViewer;
use oblikovati_martensite::planar_curve::sketch::{
    SketchDisplayEntity, SketchDisplayPrimitive, SketchSamplingOptionsV1, sample_sketch_display,
};
use oblikovati_model::theme::Rgba;
#[path = "sketch_display/blocks.rs"]
mod blocks;
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
        if self.matches(source, deflection, palette, selected) {
            return Ok(false);
        }
        let replacement = Self::replacement(viewer, source, deflection, palette, selected)?;
        replacement.show();
        *self = replacement;
        Ok(true)
    }
    fn matches(
        &self,
        source: &[ViewportSketchDisplay],
        deflection: f64,
        palette: &crate::ViewportSketchPalette,
        selected: &crate::ViewportSketchSelection,
    ) -> bool {
        self.source.as_ref().is_some_and(|(previous, error)| {
            previous == source
                && *error == deflection
                && self.color == palette.geometry
                && self.selected_color == palette.selected
                && self.selected == *selected
        })
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
        for block in &sketch.blocks {
            self.append_block(viewer, sketch, block, deflection);
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
        self.append_entity_for(viewer, sketch, entity, entity.entity, None);
    }
    fn append_entity_for(
        &mut self,
        viewer: &V3dViewer,
        sketch: &ViewportSketchDisplay,
        entity: &SketchDisplayEntity,
        identity: oblikovati_model::sketch::SketchEntityId,
        instance_color: Option<&oblikovati_api::types::Color>,
    ) {
        let selected = self.selected.sketches.contains(&sketch.sketch)
            || self.selected.entities.contains(&(sketch.sketch, identity));
        let color = entity_color(
            sketch,
            entity.entity,
            selected,
            instance_color,
            &self.color,
            &self.selected_color,
        );
        match build_entity(viewer, sketch, entity, &color, false) {
            Ok(structure) => self.structures.push(structure),
            Err(cause) => self.declines.push(display_decline(sketch, identity, cause)),
        }
    }
    pub(super) fn clear(&mut self) {
        self.structures.clear();
        self.declines.clear();
        self.source = None;
    }
}
fn entity_color(
    sketch: &ViewportSketchDisplay,
    entity: oblikovati_model::sketch::SketchEntityId,
    selected: bool,
    instance_color: Option<&oblikovati_api::types::Color>,
    default: &Rgba,
    selection: &Rgba,
) -> Rgba {
    if selected {
        return selection.clone();
    }
    if let Some(color) = instance_color {
        return api_color(color);
    }
    if let Some(color) = sketch
        .geometry
        .entity_format_overrides()
        .get(&entity)
        .map(|format| &format.override_color)
        .filter(|color| color.source == oblikovati_api::types::OVERRIDE_COLOR_SOURCE)
    {
        return api_color(color);
    }
    sketch.line_color.clone().unwrap_or_else(|| default.clone())
}
fn api_color(color: &oblikovati_api::types::Color) -> Rgba {
    Rgba {
        r: f32::from(color.r) / 255.0,
        g: f32::from(color.g) / 255.0,
        b: f32::from(color.b) / 255.0,
        a: color.opacity as f32,
    }
}
pub(super) fn build_entity(
    viewer: &V3dViewer,
    sketch: &ViewportSketchDisplay,
    entity: &SketchDisplayEntity,
    color: &Rgba,
    transient: bool,
) -> Result<SketchStructure, String> {
    let kind = entity_line_style(entity);
    match &entity.primitive {
        SketchDisplayPrimitive::Point(point) => {
            // Circle and arc centers read as center points whether or not a role was saved.
            let cross = sketch.geometry.is_center_point(entity.entity)
                || sketch.geometry.is_implicit_circular_center(entity.entity);
            let glyph = super::sketch_marker::SketchPointGlyph::from_center_role(cross);
            sketch_structure::point(viewer, sketch.plane, *point, glyph, color, transient)
        }
        SketchDisplayPrimitive::Curve(points) => {
            sketch_structure::curve(viewer, sketch.plane, points, color, kind, transient)
        }
    }
}
impl super::RenderView {
    pub(crate) fn sketch_display(
        &mut self,
        source: &[ViewportSketchDisplay],
    ) -> Result<bool, String> {
        let deflection = self.sketch_deflection()?;
        if self.sketches.matches(
            source,
            deflection,
            &self.sketch_palette,
            &self.sketch_selection,
        ) {
            return Ok(false);
        }
        let hover = self.sketch_hover_for_source(source)?;
        let changed = self.sketches.submit(
            &self.viewer,
            source,
            deflection,
            &self.sketch_palette,
            &self.sketch_selection,
        )?;
        self.replace_sketch_hover(hover);
        Ok(changed)
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
    #[test]
    fn entity_color_overrides_sketch_color_but_selection_wins() {
        use oblikovati_api::types::{Color, OVERRIDE_COLOR_SOURCE, SketchEntityFormat};
        use oblikovati_model::{
            document_sketch::DocumentSketchId,
            sketch::{Sketch2D, SketchEntityId, SketchPlane},
        };
        let id = SketchEntityId(7);
        let mut geometry = Sketch2D::default();
        geometry.add_point(id, [0.0, 0.0]).unwrap();
        let format = SketchEntityFormat {
            override_color: Color {
                r: 255,
                g: 0,
                b: 0,
                opacity: 1.0,
                source: OVERRIDE_COLOR_SOURCE,
            },
            ..Default::default()
        };
        geometry.set_entity_format(id, format).unwrap();
        let blue = Rgba {
            r: 0.0,
            g: 0.0,
            b: 1.0,
            a: 1.0,
        };
        let red = Rgba {
            r: 1.0,
            g: 0.0,
            b: 0.0,
            a: 1.0,
        };
        let green = Rgba {
            r: 0.0,
            g: 1.0,
            b: 0.0,
            a: 1.0,
        };
        let sketch = ViewportSketchDisplay {
            sketch: DocumentSketchId(1),
            plane: SketchPlane::xy(),
            geometry,
            blocks: Vec::new(),
            line_color: Some(blue.clone()),
        };
        assert_eq!(entity_color(&sketch, id, false, None, &green, &blue), red);
        assert_eq!(entity_color(&sketch, id, true, None, &green, &blue), blue);
        let override_color = Color {
            r: 0,
            g: 255,
            b: 0,
            opacity: 1.0,
            source: OVERRIDE_COLOR_SOURCE,
        };
        assert_eq!(
            entity_color(&sketch, id, false, Some(&override_color), &green, &blue),
            green
        );
        assert_eq!(
            entity_color(&sketch, id, true, Some(&override_color), &green, &blue),
            blue
        );
        assert_eq!(
            entity_color(&sketch, SketchEntityId(8), false, None, &green, &red),
            blue
        );
    }
}
