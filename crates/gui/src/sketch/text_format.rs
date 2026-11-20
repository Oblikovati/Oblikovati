// SPDX-License-Identifier: GPL-2.0-only
use super::{
    GuiSketchTextDraft,
    text_number::{HEIGHT, TextNumberInput},
};
use imgui::Ui;
use oblikovati_api::types::{
    TEXT_ALIGN_BASELINE, TEXT_ALIGN_CENTER, TEXT_ALIGN_LEFT, TEXT_ALIGN_LOWER, TEXT_ALIGN_MIDDLE,
    TEXT_ALIGN_RIGHT, TEXT_ALIGN_UPPER,
};

const SYSTEM_DEFAULT: &str = "System default";
const HEIGHT_PRESETS_MM: [&str; 6] = ["1.50", "2.50", "3.05", "4.00", "5.00", "6.10"];

pub(super) fn draw(
    ui: &Ui,
    draft: &mut GuiSketchTextDraft,
    families: &[String],
    filter: &mut String,
    height_input: &mut TextNumberInput,
) -> bool {
    font_row(ui, &mut draft.family, families, filter)
        | height_row(ui, &mut draft.height_mm, height_input)
        | ui.input_text("Rotation (deg)", &mut draft.rotation_degrees)
            .build()
        | alignment_rows(ui, draft)
}

fn font_row(ui: &Ui, family: &mut String, families: &[String], filter: &mut String) -> bool {
    ui.set_next_item_width(145.);
    let typed = ui
        .input_text("Font family", family)
        .hint(SYSTEM_DEFAULT)
        .build();
    ui.same_line();
    ui.set_next_item_width(145.);
    let Some(_list) = ui.begin_combo("##font-family-list", "Choose font") else {
        filter.clear();
        return typed;
    };
    ui.input_text("Filter fonts", filter).build();
    let search = filter.to_lowercase();
    if ui
        .selectable_config(SYSTEM_DEFAULT)
        .selected(family.is_empty())
        .build()
    {
        family.clear();
        return true;
    }
    for name in families
        .iter()
        .filter(|name| name.to_lowercase().contains(&search))
    {
        if ui.selectable_config(name).selected(name == family).build() {
            family.clone_from(name);
            return true;
        }
    }
    typed
}

fn height_row(ui: &Ui, height_mm: &mut String, input: &mut TextNumberInput) -> bool {
    let typed = input.draw(ui, height_mm, &HEIGHT);
    ui.same_line();
    ui.set_next_item_width(145.);
    let Some(_list) = ui.begin_combo("##font-height-list", "Choose height") else {
        return typed;
    };
    for preset in HEIGHT_PRESETS_MM {
        let selected = height_mm.parse::<f64>().ok() == preset.parse::<f64>().ok();
        if ui
            .selectable_config(format!("{preset} mm"))
            .selected(selected)
            .build()
        {
            input.choose_preset(height_mm, preset);
            return true;
        }
    }
    typed
}

fn alignment_rows(ui: &Ui, draft: &mut GuiSketchTextDraft) -> bool {
    let horizontal = [
        (TEXT_ALIGN_LEFT, "Left"),
        (TEXT_ALIGN_CENTER, "Center"),
        (TEXT_ALIGN_RIGHT, "Right"),
    ];
    let vertical = [
        (TEXT_ALIGN_BASELINE, "Baseline"),
        (TEXT_ALIGN_LOWER, "Lower"),
        (TEXT_ALIGN_MIDDLE, "Middle"),
        (TEXT_ALIGN_UPPER, "Upper"),
    ];
    alignment_row(ui, "Horizontal alignment", &mut draft.h_align, &horizontal)
        | alignment_row(ui, "Vertical alignment", &mut draft.v_align, &vertical)
}

fn alignment_row<T: Clone + PartialEq>(
    ui: &Ui,
    label: &str,
    current: &mut T,
    choices: &[(T, &str)],
) -> bool {
    let shown = choices
        .iter()
        .find(|(value, _)| value == current)
        .map_or("", |(_, name)| *name);
    let Some(_list) = ui.begin_combo(label, shown) else {
        return false;
    };
    for (value, name) in choices {
        if ui
            .selectable_config(name)
            .selected(value == current)
            .build()
        {
            *current = value.clone();
            return true;
        }
    }
    false
}
