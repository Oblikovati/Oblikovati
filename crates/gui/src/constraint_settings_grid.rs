// SPDX-License-Identifier: GPL-2.0-only
//! Draw the inference choices in the documented icon grid.
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use imgui::Ui;
use oblikovati_api::types::SketchInferenceTypes;

const GLYPHS: [IconGlyph; 9] = [
    IconGlyph::Horizontal,
    IconGlyph::Vertical,
    IconGlyph::Parallel,
    IconGlyph::Perpendicular,
    IconGlyph::Intersection,
    IconGlyph::Midpoint,
    IconGlyph::OnCurve,
    IconGlyph::Tangent,
    IconGlyph::Coincident,
];

pub(super) fn inference_mask(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    mut mask: u32,
    all: u32,
    entries: [(&str, SketchInferenceTypes); 9],
) -> u32 {
    if let Some(table) = crate::widget_abi::begin_table(ui, "constraint-inference-grid", 3) {
        for ((label, bit), glyph) in entries.into_iter().zip(GLYPHS) {
            ui.table_next_column();
            draw_choice(ui, icons, label, bit.0, glyph, &mut mask);
        }
        table.end();
    }
    if crate::widget_abi::button(ui, "Select All##inference", [0.; 2]) {
        mask = all;
    }
    ui.same_line();
    if crate::widget_abi::button(ui, "Clear All##inference", [0.; 2]) {
        mask = 0;
    }
    mask
}

fn draw_choice(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    label: &str,
    bit: u32,
    glyph: IconGlyph,
    mask: &mut u32,
) {
    let key = icons.key(glyph, IconSize::Small);
    crate::widget_abi::image(ui, key.raw(), [16., 16.]);
    ui.same_line();
    super::toggle_bit(ui, label, mask, bit);
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn every_inference_choice_has_a_distinct_icon() {
        let glyphs: std::collections::BTreeSet<_> = GLYPHS.into_iter().collect();
        assert_eq!(glyphs.len(), super::super::INFERENCE_TYPES.len());
    }
}
