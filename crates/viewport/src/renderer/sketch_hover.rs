// SPDX-License-Identifier: GPL-2.0-only
//! Transient feedback for sketch entities without changing selection.
use super::{RenderView, sketch_display::build_entity, sketch_structure};
use crate::{ViewportSketchDisplay, ViewportSketchFeedback, ViewportSketchHover};
use martensite_tkservice::graphic3d::Graphic3dStructureSource;
use oblikovati_martensite::planar_curve::sketch::{SketchSamplingOptionsV1, sample_sketch_entity};

#[derive(Default)]
pub(super) struct SketchHoverOverlay {
    hover: Option<ViewportSketchFeedback>,
    structures: Vec<sketch_structure::SketchStructure>,
}

impl RenderView {
    pub(crate) fn set_sketch_hover(
        &mut self,
        hover: Option<&ViewportSketchHover>,
    ) -> Result<bool, String> {
        self.set_sketch_feedback(
            hover
                .cloned()
                .map(ViewportSketchFeedback::Entities)
                .as_ref(),
        )
    }
    pub(crate) fn set_sketch_feedback(
        &mut self,
        hover: Option<&ViewportSketchFeedback>,
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
        hover: Option<&ViewportSketchFeedback>,
        sketches: &[ViewportSketchDisplay],
        palette: &crate::ViewportSketchPalette,
    ) -> Result<SketchHoverOverlay, String> {
        let Some(hover) = hover else {
            return Ok(SketchHoverOverlay::default());
        };
        let sketch = find_sketch(sketches, hover)?;
        let structures = self.hover_structures(sketch, hover, palette)?;
        Ok(SketchHoverOverlay {
            hover: Some(hover.clone()),
            structures,
        })
    }
    fn hover_structures(
        &self,
        sketch: &ViewportSketchDisplay,
        hover: &ViewportSketchFeedback,
        palette: &crate::ViewportSketchPalette,
    ) -> Result<Vec<sketch_structure::SketchStructure>, String> {
        match hover {
            ViewportSketchFeedback::Entities(ViewportSketchHover { entities, .. }) => entities
                .iter()
                .map(|id| self.hover_entity(sketch, *id, palette))
                .collect(),
            ViewportSketchFeedback::Point { position, .. } => {
                let glyph = super::sketch_marker::SketchPointGlyph::from_center_role(false);
                sketch_structure::point(
                    &self.viewer,
                    sketch.plane,
                    *position,
                    glyph,
                    &palette.candidate,
                    true,
                )
                .map(|point| vec![point])
            }
        }
    }
    fn hover_entity(
        &self,
        sketch: &ViewportSketchDisplay,
        id: oblikovati_model::sketch::SketchEntityId,
        palette: &crate::ViewportSketchPalette,
    ) -> Result<sketch_structure::SketchStructure, String> {
        let entity = sample_sketch_entity(
            &sketch.geometry,
            id,
            SketchSamplingOptionsV1 {
                deflection: self.sketch_deflection()?,
                maximum_curve_points: 4096,
            },
        )?;
        build_entity(&self.viewer, sketch, &entity, &palette.candidate, true)
    }
}

fn hover_still_valid(sketches: &[ViewportSketchDisplay], hover: &ViewportSketchFeedback) -> bool {
    let ViewportSketchFeedback::Entities(ViewportSketchHover { entities, .. }) = hover else {
        // Derived positions must be recomputed after any source change.
        return false;
    };
    find_sketch(sketches, hover).is_ok_and(|sketch| {
        entities
            .iter()
            .all(|id| sketch.geometry.entity_kind(*id).is_ok())
    })
}

fn find_sketch<'a>(
    sketches: &'a [ViewportSketchDisplay],
    hover: &ViewportSketchFeedback,
) -> Result<&'a ViewportSketchDisplay, String> {
    let id = match hover {
        ViewportSketchFeedback::Entities(ViewportSketchHover { sketch, entities }) => {
            let mut distinct = std::collections::BTreeSet::new();
            if entities.is_empty() || !entities.iter().all(|id| distinct.insert(*id)) {
                return Err(format!(
                    "sketch hover {entities:?}: expected distinct entity identities"
                ));
            }
            *sketch
        }
        ViewportSketchFeedback::Point { sketch, position } => {
            if !position.iter().all(|coordinate| coordinate.is_finite()) {
                return Err(format!(
                    "sketch hover point {position:?}: expected finite coordinates"
                ));
            }
            *sketch
        }
    };
    let mut found = sketches.iter().filter(|sketch| sketch.sketch == id);
    let sketch = found
        .next()
        .ok_or_else(|| format!("hover sketch {id:?}: expected one displayed sketch"))?;
    if found.next().is_some() {
        return Err(format!(
            "hover sketch {id:?}: expected an unambiguous display"
        ));
    }
    Ok(sketch)
}
