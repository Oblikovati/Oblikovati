// SPDX-License-Identifier: GPL-2.0-only
//! ImGui widget rendering for Part Modify Shell v2 (Behavior, Override Sets, Advanced Properties, #4106).
use super::{
    GuiShellOverrideSet, GuiShellPickMode, ShellV2State, parse_positive_mm, parse_positive_percent,
};
use crate::GuiShellAction;
use imgui::{StyleColor, StyleVar, Ui};
use oblikovati_model::feature::shell::{
    ShellApproximationTolerance, ShellApproximationType, ShellDirection, ShellType,
};

const ACTIVE_BLUE: [f32; 4] = [0.24, 0.45, 0.78, 1.0];
const PICK_TEAL: [f32; 4] = [0.18, 0.56, 0.62, 1.0];
const ERROR_RED: [f32; 4] = [0.92, 0.24, 0.24, 1.0];

pub(super) fn draw_shell_body(
    v2: &mut ShellV2State,
    ui: &Ui,
    thickness: &mut String,
    actions: &mut Vec<GuiShellAction>,
) {
    draw_behavior_section(v2, ui, thickness, actions);
    if v2.shell_type == ShellType::Sharp {
        ui.separator();
        draw_override_sets_section(v2, ui, actions);
        ui.separator();
        draw_advanced_properties_section(v2, ui);
    }
}

fn draw_behavior_section(
    v2: &mut ShellV2State,
    ui: &Ui,
    thickness: &mut String,
    actions: &mut Vec<GuiShellAction>,
) {
    ui.text_disabled("Behavior");
    crate::theme::property_label(ui, "Shell Type");
    let types = [(ShellType::Sharp, "Sharp"), (ShellType::Rounded, "Rounded")];
    for (idx, (kind, label)) in types.into_iter().enumerate() {
        if idx > 0 {
            ui.same_line_with_spacing(0.0, 4.0);
        }
        let active = v2.shell_type == kind;
        let _bg = active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
        if ui.button_with_size(format!("{label}##shell-type-{idx}"), [88.0, 0.0]) && !active {
            v2.shell_type = kind;
            if kind == ShellType::Rounded && matches!(v2.pick_mode, GuiShellPickMode::Override(_)) {
                v2.set_pick_mode(GuiShellPickMode::Remove, actions);
            }
        }
    }

    crate::theme::property_label(ui, "Direction");
    let dirs = [
        (ShellDirection::Inside, "Inside"),
        (ShellDirection::Outside, "Outside"),
        (ShellDirection::Both, "Both"),
    ];
    for (idx, (dir, label)) in dirs.into_iter().enumerate() {
        if idx > 0 {
            ui.same_line_with_spacing(0.0, 4.0);
        }
        let active = v2.direction == dir;
        let _bg = active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
        if ui.button_with_size(format!("{label}##shell-dir-{idx}"), [60.0, 0.0]) && !active {
            v2.direction = dir;
        }
    }

    crate::theme::property_label(ui, "Remove");
    let count = v2.openings.len();
    let remove_label = match count {
        0 => "Select Faces".to_owned(),
        1 => "1 Face".to_owned(),
        n => format!("{n} Faces"),
    };
    let remove_active = v2.pick_mode == GuiShellPickMode::Remove;
    let _bg = remove_active.then(|| ui.push_style_color(StyleColor::Button, PICK_TEAL));
    if ui.button(format!("{remove_label}##shell-remove-pick")) {
        v2.set_pick_mode(GuiShellPickMode::Remove, actions);
    }

    let invalid_thickness = parse_positive_mm(thickness).is_none();
    crate::theme::property_label(ui, "Thickness");
    {
        let _w = ui.push_item_width(-22.0);
        let _numeric = crate::ui_scale::numeric_font(ui);
        let _border_w = ui.push_style_var(StyleVar::FrameBorderSize(1.0));
        let _text_c = invalid_thickness.then(|| ui.push_style_color(StyleColor::Text, ERROR_RED));
        let _border_c =
            invalid_thickness.then(|| ui.push_style_color(StyleColor::Border, ERROR_RED));
        let _ = ui.input_text("##shell-thickness", thickness).build();
    }
    if invalid_thickness {
        ui.same_line_with_spacing(0.0, 4.0);
        ui.text_colored(ERROR_RED, "!");
    }
}

fn draw_override_sets_section(v2: &mut ShellV2State, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
    ui.text_disabled("Thickness Override Sets");
    ui.same_line_with_spacing(0.0, 8.0);
    if ui.button("+##shell-add-override") {
        let index = v2.overrides.len();
        v2.overrides.push(GuiShellOverrideSet {
            face: None,
            thickness: "1 mm".into(),
        });
        v2.set_pick_mode(GuiShellPickMode::Override(index), actions);
    }
    let mut activate_override = None;
    for (idx, ov) in v2.overrides.iter_mut().enumerate() {
        let active = v2.pick_mode == GuiShellPickMode::Override(idx);
        let pill = if ov.face.is_some() {
            "1 Face"
        } else if active {
            "Pick Face..."
        } else {
            "Select Face"
        };
        let _bg = active.then(|| ui.push_style_color(StyleColor::Button, PICK_TEAL));
        if ui.button(format!("{pill}##shell-ov-face-{idx}")) {
            activate_override = Some(idx);
        }
        drop(_bg);
        ui.same_line_with_spacing(0.0, 6.0);
        let invalid = ov.face.is_some() && parse_positive_mm(&ov.thickness).is_none();
        let _w = ui.push_item_width(110.0);
        let _numeric = crate::ui_scale::numeric_font(ui);
        let _border_w = ui.push_style_var(StyleVar::FrameBorderSize(1.0));
        let _text_c = invalid.then(|| ui.push_style_color(StyleColor::Text, ERROR_RED));
        let _border_c = invalid.then(|| ui.push_style_color(StyleColor::Border, ERROR_RED));
        let _ = ui
            .input_text(format!("##shell-ov-thick-{idx}"), &mut ov.thickness)
            .build();
    }
    if let Some(idx) = activate_override {
        v2.set_pick_mode(GuiShellPickMode::Override(idx), actions);
    }
}

fn draw_advanced_properties_section(v2: &mut ShellV2State, ui: &Ui) {
    ui.text_disabled("Advanced Properties");
    ui.checkbox(
        "Allow Approximation##shell-approx",
        &mut v2.allow_approximation,
    );

    let _disabled = ui.begin_disabled(!v2.allow_approximation);
    crate::theme::property_label(ui, "Type");
    let choices = [
        (ShellApproximationType::NeverTooThin, "Never too thin"),
        (ShellApproximationType::NeverTooThick, "Never too thick"),
        (ShellApproximationType::Mean, "Mean"),
    ];
    let current_label = choices
        .iter()
        .find(|(kind, _)| *kind == v2.approximation_type)
        .map_or("Mean", |(_, label)| *label);
    if let Some(_combo) = ui.begin_combo("##shell-approx-type", current_label) {
        for (kind, label) in choices {
            let selected = v2.approximation_type == kind;
            if ui.selectable_config(label).selected(selected).build() {
                v2.approximation_type = kind;
            }
        }
    }

    crate::theme::property_label(ui, "Tolerance");
    let tol_choices = [
        (ShellApproximationTolerance::Optimized, "Optimized"),
        (
            ShellApproximationTolerance::Specified,
            "Specified tolerance",
        ),
    ];
    for (idx, (tol, label)) in tol_choices.into_iter().enumerate() {
        if idx > 0 {
            ui.same_line_with_spacing(0.0, 4.0);
        }
        let active = v2.approximation_tolerance == tol;
        let _bg = active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
        if ui.button(format!("{label}##shell-tol-{idx}")) {
            v2.approximation_tolerance = tol;
        }
    }

    if v2.approximation_tolerance == ShellApproximationTolerance::Specified {
        let invalid_pct = parse_positive_percent(&v2.approximation_percent).is_none();
        crate::theme::property_label(ui, "Percent");
        let _numeric = crate::ui_scale::numeric_font(ui);
        let _border_w = ui.push_style_var(StyleVar::FrameBorderSize(1.0));
        let _text_c = invalid_pct.then(|| ui.push_style_color(StyleColor::Text, ERROR_RED));
        let _border_c = invalid_pct.then(|| ui.push_style_color(StyleColor::Border, ERROR_RED));
        let _ = ui
            .input_text("##shell-approx-percent", &mut v2.approximation_percent)
            .build();
    }
}
