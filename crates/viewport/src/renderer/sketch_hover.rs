// SPDX-License-Identifier: GPL-2.0-only
//! A small transient overlay for the lines named by a hovered constraint.
use super::{RenderView, sketch_display::curve_role_line_style, sketch_structure};
use crate::{ViewportSketchDisplay, ViewportSketchHover};
use martensite_tkservice::graphic3d::Graphic3dStructureSource;

#[derive(Default)]
pub(super) struct SketchHoverOverlay {
    hover: Option<ViewportSketchHover>,
    structures: Vec<sketch_structure::SketchStructure>,
}

impl RenderView {
    pub(crate) fn set_sketch_hover(
        &mut self,
        hover: Option<&ViewportSketchHover>,
    ) -> Result<bool, String> {
        if self.sketch_hover.hover.as_ref() == hover {
            return Ok(false);
        }
        let source = self
            .sketches
            .source
            .as_ref()
            .map_or(&[][..], |(rows, _)| rows.as_slice());
        let replacement = self.build_sketch_hover(hover, source, &self.sketch_palette)?;
        self.replace_sketch_hover(replacement);
        Ok(true)
    }
    pub(super) fn replace_sketch_hover(&mut self, replacement: SketchHoverOverlay) {
        for structure in &replacement.structures {
            structure.0.display();
        }
        self.sketch_hover = replacement;
    }
    pub(super) fn sketch_hover_for_source(
        &self,
        source: &[ViewportSketchDisplay],
    ) -> Result<SketchHoverOverlay, String> {
        let hover = self
            .sketch_hover
            .hover
            .as_ref()
            .filter(|hover| hover_still_valid(source, hover));
        self.build_sketch_hover(hover, source, &self.sketch_palette)
    }
    pub(super) fn sketch_hover_for_palette(
        &self,
        palette: &crate::ViewportSketchPalette,
    ) -> Result<SketchHoverOverlay, String> {
        let source = self
            .sketches
            .source
            .as_ref()
            .map_or(&[][..], |(rows, _)| rows.as_slice());
        self.build_sketch_hover(self.sketch_hover.hover.as_ref(), source, palette)
    }
    pub(super) fn build_sketch_hover(
        &self,
        hover: Option<&ViewportSketchHover>,
        sketches: &[ViewportSketchDisplay],
        palette: &crate::ViewportSketchPalette,
    ) -> Result<SketchHoverOverlay, String> {
        let Some(hover) = hover else {
            return Ok(SketchHoverOverlay::default());
        };
        let sketch = find_sketch(sketches, hover)?;
        let mut structures = Vec::with_capacity(hover.entities.len());
        for id in &hover.entities {
            structures.push(self.hover_line(sketch, *id, palette)?);
        }
        Ok(SketchHoverOverlay {
            hover: Some(hover.clone()),
            structures,
        })
    }
    fn hover_line(
        &self,
        sketch: &ViewportSketchDisplay,
        id: oblikovati_model::sketch::SketchEntityId,
        palette: &crate::ViewportSketchPalette,
    ) -> Result<sketch_structure::SketchStructure, String> {
        let line = sketch
            .geometry
            .line(id)
            .map_err(|error| error.to_string())?;
        let [a, b] = line.endpoints;
        let a = sketch
            .geometry
            .point(a)
            .map_err(|error| error.to_string())?
            .position;
        let b = sketch
            .geometry
            .point(b)
            .map_err(|error| error.to_string())?
            .position;
        let style = curve_role_line_style(
            sketch
                .geometry
                .is_construction(id)
                .map_err(|error| error.to_string())?,
            sketch
                .geometry
                .is_centerline(id)
                .map_err(|error| error.to_string())?,
        );
        sketch_structure::curve(
            &self.viewer,
            sketch.plane,
            &[a, b],
            &palette.candidate,
            style,
            true,
        )
    }
}

fn hover_still_valid(sketches: &[ViewportSketchDisplay], hover: &ViewportSketchHover) -> bool {
    find_sketch(sketches, hover).is_ok_and(|sketch| {
        hover
            .entities
            .iter()
            .all(|id| sketch.geometry.line(*id).is_ok())
    })
}

fn find_sketch<'a>(
    sketches: &'a [ViewportSketchDisplay],
    hover: &ViewportSketchHover,
) -> Result<&'a ViewportSketchDisplay, String> {
    if hover.entities.len() != 2 || hover.entities[0] == hover.entities[1] {
        return Err(format!(
            "constraint hover {:?}: expected two distinct line identities",
            hover.entities
        ));
    }
    let mut found = sketches
        .iter()
        .filter(|sketch| sketch.sketch == hover.sketch);
    let sketch = found.next().ok_or_else(|| {
        format!(
            "constraint hover sketch {:?}: expected one displayed sketch",
            hover.sketch
        )
    })?;
    if found.next().is_some() {
        return Err(format!(
            "constraint hover sketch {:?}: expected an unambiguous display",
            hover.sketch
        ));
    }
    Ok(sketch)
}
