// SPDX-License-Identifier: GPL-2.0-only
//! Disclosure and enablement for the remaining Generic appearance effects.
use imgui::Ui;
use oblikovati_model::assets::AppearanceMetadata;

pub(super) fn draw(ui: &Ui, metadata: &mut AppearanceMetadata) {
    let original = metadata.generic_effects.clone().unwrap_or_default();
    let mut effects = original.clone();
    section(ui, "Cutouts", &mut effects.cutouts_enabled, || {
        image_slot(ui);
    });
    let illumination = &mut effects.illumination;
    section(ui, "Self Illumination", &mut illumination.enabled, || {
        draw_illumination(
            ui,
            &mut illumination.color,
            &mut illumination.luminance,
            &mut illumination.temperature,
        );
    });
    section(ui, "Bump", &mut effects.bump.enabled, || {
        image_slot(ui);
        if let Some(surface) = &effects.rough_surface {
            ui.same_line_with_spacing(0., 14.);
            ui.text(
                surface
                    .bump_image
                    .as_ref()
                    .map_or("Wavy", |image| image.name()),
            );
        }
        ui.slider_config("Amount", 0, 100)
            .display_format("%d%%")
            .build(&mut effects.bump.amount);
    });
    section(ui, "Tint", &mut effects.tint.enabled, || {
        edit_effect_color(ui, "Tint Color", &mut effects.tint.color);
    });
    if effects != original {
        metadata.generic_effects = (effects != Default::default()).then_some(effects);
    }
}

pub(super) fn section(ui: &Ui, title: &str, enabled: &mut bool, controls: impl FnOnce()) {
    let _id = ui.push_id(title);
    let expanded = ui.tree_node_config(title).push();
    ui.same_line();
    ui.checkbox("Enabled", enabled);
    if expanded.is_some() {
        let _disabled = ui.begin_disabled(!*enabled);
        controls();
    }
}

fn draw_illumination(ui: &Ui, color: &mut [u8; 3], luminance: &mut u16, temperature: &mut u16) {
    edit_effect_color(ui, "Filter Color", color);
    preset_value(
        ui,
        "Luminance",
        luminance,
        [0, 5000],
        &[(50, "50 cd/m²"), (100, "100 cd/m²"), (500, "500 cd/m²")],
        "%d cd/m²",
    );
    preset_value(
        ui,
        "Color Temperature",
        temperature,
        [1000, 12000],
        &[(3000, "3000 K"), (6500, "6500 K"), (9000, "9000 K")],
        "%d K",
    );
}

fn image_slot(ui: &Ui) {
    let _unavailable = ui.begin_disabled(true);
    ui.button("Image");
}

fn preset_value(
    ui: &Ui,
    label: &str,
    value: &mut u16,
    [min, max]: [u16; 2],
    presets: &[(u16, &str)],
    format: &str,
) {
    let _id = ui.push_id(label);
    let preview = presets
        .iter()
        .find(|(preset, _)| *preset == *value)
        .map_or("Custom", |(_, label)| *label);
    ui.set_next_item_width(105.);
    if let Some(_choices) = ui.begin_combo("##preset", preview) {
        for &(preset, name) in presets {
            if ui.selectable(name) {
                *value = preset;
            }
        }
    }
    ui.same_line();
    ui.set_next_item_width(100.);
    ui.slider_config(label, min, max)
        .display_format(format)
        .build(value);
}

pub(super) fn edit_effect_color(ui: &Ui, label: &str, color: &mut [u8; 3]) {
    let mut components = color.map(|channel| f32::from(channel) / 255.);
    if ui.color_edit3(label, &mut components) {
        *color = components.map(|channel| (channel.clamp(0., 1.) * 255.).round() as u8);
    }
}

pub(super) fn choice<T: Copy + Eq>(ui: &Ui, label: &str, selected: &mut T, choices: &[(T, &str)]) {
    let Some((_, preview)) = choices.iter().find(|(value, _)| value == selected) else {
        return;
    };
    let Some(_menu) = ui.begin_combo(label, *preview) else {
        return;
    };
    for (value, name) in choices {
        if ui
            .selectable_config(*name)
            .selected(value == selected)
            .build()
        {
            *selected = *value;
        }
    }
}
