// SPDX-License-Identifier: GPL-2.0-only
use super::{
    GuiSketchTextDraft,
    text_content::apply_text_width,
    text_number::{TextNumberInput, WIDTH},
};
use imgui::Ui;
use oblikovati_model::sketch::uniform_sketch_text_width;

pub(super) fn draw(ui: &Ui, draft: &mut GuiSketchTextDraft, input: &mut TextNumberInput) -> bool {
    let mut value = current_width(draft);
    let _disabled = ui.begin_disabled(draft.content.is_empty() || value.is_empty());
    input.draw(ui, &mut value, &WIDTH) && update_width(draft, &value)
}

pub(super) fn apply(draft: &mut GuiSketchTextDraft, input: &mut TextNumberInput) {
    let mut value = current_width(draft);
    if input.apply(&mut value, &WIDTH) {
        update_width(draft, &value);
    }
}

fn current_width(draft: &GuiSketchTextDraft) -> String {
    let runs = draft.formatting.runs.as_deref().unwrap_or_default();
    uniform_sketch_text_width(&draft.content, runs)
        .map_or_else(String::new, |width| width.to_string())
}

fn update_width(draft: &mut GuiSketchTextDraft, value: &str) -> bool {
    value
        .trim()
        .parse()
        .is_ok_and(|percent| apply_text_width(draft, percent))
}
