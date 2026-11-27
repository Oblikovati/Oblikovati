// SPDX-License-Identifier: GPL-2.0-only
use imgui::Ui;
use oblikovati_model::assets::{AppearanceMetadata, AppearanceTransparency, RefractionPreset};

const PRESETS: [(RefractionPreset, &str); 7] = [
    (RefractionPreset::Air, "Air"),
    (RefractionPreset::Water, "Water"),
    (RefractionPreset::Alcohol, "Alcohol"),
    (RefractionPreset::Quartz, "Quartz"),
    (RefractionPreset::Glass, "Glass"),
    (RefractionPreset::Diamond, "Diamond"),
    (RefractionPreset::Custom, "Custom"),
];

pub(super) fn draw(ui: &Ui, metadata: &mut AppearanceMetadata) {
    let _id = ui.push_id("transparency");
    let original = metadata.transparency.unwrap_or_default();
    let mut effect = original;
    let expanded = ui.tree_node_config("Transparency").push();
    ui.same_line();
    ui.checkbox("Enabled", &mut effect.enabled);
    if expanded.is_some() {
        let _disabled = ui.begin_disabled(!effect.enabled);
        ui.slider_config("Amount", 0, 100)
            .display_format("%d%%")
            .build(&mut effect.amount);
        ui.text_disabled("Image");
        ui.text_disabled("Image Fade");
        ui.slider_config("Translucency", 0, 100)
            .display_format("%d%%")
            .build(&mut effect.translucency);
        draw_refraction(ui, &mut effect);
    }
    if effect != original {
        metadata.transparency = (effect != Default::default()).then_some(effect);
    }
}

fn draw_refraction(ui: &Ui, effect: &mut AppearanceTransparency) {
    let mut selected = PRESETS
        .iter()
        .position(|(preset, _)| *preset == effect.refraction)
        .unwrap_or(4);
    if ui.combo_simple_string(
        "Refraction",
        &mut selected,
        &PRESETS.map(|(_, label)| label),
    ) {
        effect.refraction = PRESETS[selected].0;
    }
    let mut index = effect.index_of_refraction();
    let _preset = ui.begin_disabled(effect.refraction != RefractionPreset::Custom);
    if ui
        .input_float("Index of refraction", &mut index)
        .display_format("%.3f")
        .build()
        && index.is_finite()
    {
        effect.custom_index_milli = (index.clamp(0.001, 5.) * 1000.).round() as u16;
    }
}
