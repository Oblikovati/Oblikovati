// SPDX-License-Identifier: GPL-2.0-only
//! Small marks beside constrained geometry, drawn while a sketch shows its constraints.
use crate::GuiFrameRequest;

/// One relation's mark at its window pixel; example: `H` beside a horizontal line.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiConstraintGlyph {
    pub pixel: [i32; 2],
    pub text: String,
}
const PADDING: f32 = 2.;
/// Beside the point it describes, so the mark never hides the geometry it marks.
const OFFSET: f32 = 8.;
/// Draw behind every panel, as dimension annotations are.
pub(crate) fn draw_constraint_glyphs(ui: &imgui::Ui, request: &GuiFrameRequest) {
    let scale = request.layout_scale();
    let background = ui.style_color(imgui::StyleColor::PopupBg);
    let text = ui.style_color(imgui::StyleColor::Text);
    let list = ui.get_background_draw_list();
    for glyph in &request.commands.constraint_glyphs {
        let size = ui.calc_text_size(&glyph.text);
        let at = [
            glyph.pixel[0] as f32 / scale[0] + OFFSET,
            glyph.pixel[1] as f32 / scale[1] + OFFSET,
        ];
        list.add_rect(
            [at[0] - PADDING, at[1] - PADDING],
            [at[0] + size[0] + PADDING, at[1] + size[1] + PADDING],
            background,
        )
        .filled(true)
        .build();
        list.add_text(at, text, &glyph.text);
    }
}
