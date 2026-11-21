// SPDX-License-Identifier: GPL-2.0-only
//! Small marks beside constrained geometry, drawn while a sketch shows its constraints.
use crate::GuiFrameRequest;

/// One relation's mark at its window pixel; example: `H` beside a horizontal line.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiConstraintGlyph {
    pub sketch: oblikovati_model::document_sketch::DocumentSketchId,
    pub constraint: oblikovati_model::sketch::SketchConstraintId,
    pub related: Vec<oblikovati_model::sketch::SketchEntityId>,
    pub pixel: [i32; 2],
    pub text: String,
}
/// Transient relation feedback; it does not change the document selection.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiConstraintHover {
    pub sketch: oblikovati_model::document_sketch::DocumentSketchId,
    pub constraint: oblikovati_model::sketch::SketchConstraintId,
    pub related: Vec<oblikovati_model::sketch::SketchEntityId>,
    pub candidates: Vec<GuiConstraintCandidate>,
}
/// An exact relation hit offered in an overlapping-glyph chooser.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiConstraintCandidate {
    pub constraint: oblikovati_model::sketch::SketchConstraintId,
    pub text: String,
}
const PADDING: f32 = 2.;
/// Beside the point it describes, so the mark never hides the geometry it marks.
const OFFSET: f32 = 8.;
/// Draw behind every panel, as dimension annotations are.
pub(crate) fn draw_constraint_glyphs(
    ui: &imgui::Ui,
    request: &GuiFrameRequest,
) -> Option<GuiConstraintHover> {
    let scale = request.layout_scale();
    let mut hits = Vec::new();
    for glyph in &request.commands.constraint_glyphs {
        if draw_glyph(ui, glyph, scale) {
            hits.push(glyph);
        }
    }
    let first = *hits.first()?;
    let mut candidates: Vec<_> = hits
        .iter()
        .filter(|glyph| glyph.sketch == first.sketch)
        .map(|glyph| GuiConstraintCandidate {
            constraint: glyph.constraint,
            text: glyph.text.clone(),
        })
        .collect();
    candidates.sort_unstable_by_key(|candidate| candidate.constraint);
    Some(GuiConstraintHover {
        sketch: first.sketch,
        constraint: first.constraint,
        related: if candidates.len() == 1 {
            first.related.clone()
        } else {
            Vec::new()
        },
        candidates,
    })
}

fn draw_glyph(ui: &imgui::Ui, glyph: &GuiConstraintGlyph, scale: [f32; 2]) -> bool {
    let size = ui.calc_text_size(&glyph.text);
    let at = [
        glyph.pixel[0] as f32 / scale[0] + OFFSET,
        glyph.pixel[1] as f32 / scale[1] + OFFSET,
    ];
    let list = ui.get_background_draw_list();
    list.add_rect(
        [at[0] - PADDING, at[1] - PADDING],
        [at[0] + size[0] + PADDING, at[1] + size[1] + PADDING],
        ui.style_color(imgui::StyleColor::PopupBg),
    )
    .filled(true)
    .build();
    list.add_text(at, ui.style_color(imgui::StyleColor::Text), &glyph.text);
    !glyph.related.is_empty() && glyph_contains(glyph.pixel, size, scale, ui.io().mouse_pos)
}

/// Find one unobstructed mark under the pointer; ambiguous marks do not select a relation.
pub(crate) fn unobstructed_hover(
    ui: &imgui::Ui,
    hover: Option<GuiConstraintHover>,
) -> Option<GuiConstraintHover> {
    if ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel())
        || crate::widget_abi::any_popup_open(ui)
    {
        return None;
    }
    hover
}

fn glyph_contains(pixel: [i32; 2], size: [f32; 2], scale: [f32; 2], mouse: [f32; 2]) -> bool {
    let at = [
        pixel[0] as f32 / scale[0] + OFFSET,
        pixel[1] as f32 / scale[1] + OFFSET,
    ];
    (0..2).all(|axis| {
        mouse[axis] >= at[axis] - PADDING && mouse[axis] < at[axis] + size[axis] + PADDING
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn hit_region_matches_scaled_drawn_glyph_and_padding() {
        assert!(glyph_contains([100, 80], [12., 10.], [2., 2.], [58., 48.]));
        assert!(glyph_contains([100, 80], [12., 10.], [2., 2.], [56., 46.]));
        assert!(!glyph_contains([100, 80], [12., 10.], [2., 2.], [72., 48.]));
        assert!(!glyph_contains([100, 80], [12., 10.], [2., 2.], [55., 48.]));
    }
}
