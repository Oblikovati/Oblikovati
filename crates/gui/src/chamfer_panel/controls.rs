// SPDX-License-Identifier: GPL-2.0-only
//! ImGui widget rendering for Part Modify Chamfer v2 (Chamfer & Partial tabs, modes, error modal, #4046).
use super::{
    ChamferV2State, GuiChamferPickMode, GuiChamferTab, GuiChamferV2Action, parse_angle_deg,
    parse_positive_mm,
};
use crate::{GuiCommandState, GuiShellAction};
use imgui::{StyleColor, StyleVar, Ui};
use oblikovati_model::feature::chamfer_options::{ChamferDrivenDimension, ChamferMode};

const ERROR_RED: [f32; 4] = [0.92, 0.24, 0.24, 1.0];

pub(super) fn draw_chamfer_body(
    v2: &mut ChamferV2State,
    ui: &Ui,
    state: &GuiCommandState,
    distance: &mut String,
    is_edit: bool,
    edited_edges: usize,
    actions: &mut Vec<GuiShellAction>,
) {
    draw_tab_bar(v2, ui, actions);
    ui.separator();
    match v2.tab {
        GuiChamferTab::Chamfer => {
            draw_chamfer_tab(v2, ui, state, distance, is_edit, edited_edges, actions);
        }
        GuiChamferTab::Partial => {
            draw_partial_tab(v2, ui, state);
        }
    }
}

fn draw_tab_bar(v2: &mut ChamferV2State, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
    let tabs = [
        (GuiChamferTab::Chamfer, "Chamfer"),
        (GuiChamferTab::Partial, "Partial"),
    ];
    for (idx, (tab, label)) in tabs.into_iter().enumerate() {
        if idx > 0 {
            ui.same_line_with_spacing(0.0, 4.0);
        }
        let active = v2.tab == tab;
        let _bg = active.then(|| ui.push_style_color(StyleColor::Button, [0.24, 0.45, 0.78, 1.0]));
        if ui.button_with_size(format!("{label}##chamfer-tab-{idx}"), [120.0, 0.0]) && !active {
            v2.tab = tab;
            let pick_mode = match tab {
                GuiChamferTab::Partial => GuiChamferPickMode::PartialEdge,
                GuiChamferTab::Chamfer if v2.face_pick_active => GuiChamferPickMode::Face,
                GuiChamferTab::Chamfer => GuiChamferPickMode::None,
            };
            actions.push(GuiShellAction::ChamferV2(GuiChamferV2Action::SetPickMode(
                pick_mode,
            )));
        }
    }
}

fn draw_chamfer_tab(
    v2: &mut ChamferV2State,
    ui: &Ui,
    state: &GuiCommandState,
    distance: &mut String,
    is_edit: bool,
    edited_edges: usize,
    actions: &mut Vec<GuiShellAction>,
) {
    let modes = [
        (ChamferMode::EqualDistance, "Equal"),
        (ChamferMode::DistanceAndAngle, "Dist/Angle"),
        (ChamferMode::TwoDistances, "Two Dist"),
    ];
    for (idx, (mode, label)) in modes.into_iter().enumerate() {
        if idx > 0 {
            ui.same_line_with_spacing(0.0, 4.0);
        }
        let active = v2.mode == mode;
        let _bg = active.then(|| ui.push_style_color(StyleColor::Button, [0.24, 0.45, 0.78, 1.0]));
        if ui.button_with_size(format!("{label}##chamfer-mode-{idx}"), [82.0, 0.0]) && !active {
            v2.mode = mode;
            if mode != ChamferMode::DistanceAndAngle && v2.face_pick_active {
                v2.face_pick_active = false;
                actions.push(GuiShellAction::ChamferV2(GuiChamferV2Action::SetPickMode(
                    GuiChamferPickMode::None,
                )));
            }
        }
    }
    ui.separator();

    let count = if is_edit {
        edited_edges.max(v2.edges.len())
    } else {
        v2.edges.len().max(state.selected_edge_count)
    };
    let selection = match count {
        0 => "Select Edges".to_owned(),
        1 => "1 Edge".to_owned(),
        n => format!("{n} Edges"),
    };
    crate::theme::property_label(ui, "Edges");
    crate::feature_properties::draw_pick_pill(ui, &selection, count > 0);

    if v2.mode == ChamferMode::DistanceAndAngle {
        crate::theme::property_label(ui, "Face");
        let face_label = if v2.reference_face.is_some() {
            "1 Face"
        } else if v2.face_pick_active {
            "Pick Face..."
        } else {
            "Select Face"
        };
        let _bg = v2
            .face_pick_active
            .then(|| ui.push_style_color(StyleColor::Button, [0.18, 0.56, 0.62, 1.0]));
        if ui.button(format!("{face_label}##chamfer-face-pick")) {
            v2.face_pick_active = !v2.face_pick_active;
            let mode = if v2.face_pick_active {
                GuiChamferPickMode::Face
            } else {
                GuiChamferPickMode::None
            };
            actions.push(GuiShellAction::ChamferV2(GuiChamferV2Action::SetPickMode(
                mode,
            )));
        }
    }

    match v2.mode {
        ChamferMode::EqualDistance => {
            draw_length_row(ui, "Distance", "##blend-value", distance);
        }
        ChamferMode::DistanceAndAngle => {
            draw_length_row(ui, "Distance", "##blend-value", distance);
            draw_angle_row(ui, "Angle", "##chamfer-angle", &mut v2.angle);
        }
        ChamferMode::TwoDistances => {
            draw_length_row(ui, "Distance1", "##blend-value", distance);
            draw_length_row(ui, "Distance2", "##chamfer-dist2", &mut v2.distance2);
            crate::theme::property_label(ui, "Direction");
            if ui.button("Reverse##chamfer-reverse") {
                std::mem::swap(distance, &mut v2.distance2);
                v2.reversed = !v2.reversed;
            }
        }
    }

    ui.separator();
    crate::theme::property_label(ui, "Edge Chain");
    draw_pair_toggle(
        ui,
        "Chain",
        "Single",
        "##chamfer-chain",
        &mut v2.edge_chain_second,
    );
    crate::theme::property_label(ui, "Setback");
    draw_pair_toggle(
        ui,
        "Sharp",
        "Flat",
        "##chamfer-setback",
        &mut v2.setback_second,
    );
}

fn draw_partial_tab(v2: &mut ChamferV2State, ui: &Ui, state: &GuiCommandState) {
    crate::theme::property_label(ui, "Set Driven Dimension");
    let choices = [
        (ChamferDrivenDimension::ToStart, "To Start"),
        (ChamferDrivenDimension::Chamfer, "Chamfer"),
        (ChamferDrivenDimension::ToEnd, "To End"),
    ];
    let current = choices
        .iter()
        .find(|(d, _)| *d == v2.driven_choice)
        .map_or("Chamfer", |(_, l)| *l);
    for (idx, (dim, label)) in choices.into_iter().enumerate() {
        if idx > 0 {
            ui.same_line_with_spacing(0.0, 4.0);
        }
        let active = v2.driven_choice == dim;
        let _bg = active.then(|| ui.push_style_color(StyleColor::Button, [0.24, 0.45, 0.78, 1.0]));
        if ui.button(format!("{label}##driven-{idx}")) && !active {
            v2.driven_choice = dim;
            v2.recalculate_partial();
        }
    }
    let _ = current;
    ui.separator();

    let sel_text = if v2.partial.is_some() {
        "1 Selected"
    } else {
        "0 Selected"
    };
    ui.text(format!("Partial Edges: {sel_text}"));
    if v2.partial.is_none() && !v2.edges.is_empty() {
        ui.same_line_with_spacing(0.0, 8.0);
        if ui.button("Add Edge##partial-add") {
            v2.activate_partial_from_edge(state);
        }
    }
    if let Some(partial) = v2.partial.as_mut() {
        let mut changed = false;
        let is_start_driven = partial.driven == ChamferDrivenDimension::ToStart;
        let is_chamfer_driven = partial.driven == ChamferDrivenDimension::Chamfer;
        let is_end_driven = partial.driven == ChamferDrivenDimension::ToEnd;

        changed |= draw_partial_row(
            ui,
            "To Start",
            "##partial-start",
            &mut partial.to_start,
            is_start_driven,
        );
        changed |= draw_partial_row(
            ui,
            "Chamfer",
            "##partial-len",
            &mut partial.chamfer_length,
            is_chamfer_driven,
        );
        changed |= draw_partial_row(
            ui,
            "To End",
            "##partial-end",
            &mut partial.to_end,
            is_end_driven,
        );
        if changed {
            v2.recalculate_partial();
        }
    } else {
        ui.text_disabled("To Start | Chamfer | To End");
    }
}

fn draw_partial_row(ui: &Ui, label: &str, id: &str, value: &mut String, driven: bool) -> bool {
    crate::theme::property_label(ui, label);
    let _numeric = crate::ui_scale::numeric_font(ui);
    let _border = ui.push_style_var(StyleVar::FrameBorderSize(1.0));
    let _disabled = ui.begin_disabled(driven);
    ui.input_text(id, value).build()
}

fn draw_length_row(ui: &Ui, label: &str, id: &str, value: &mut String) {
    let invalid = parse_positive_mm(value).is_none();
    crate::theme::property_label(ui, label);
    let _numeric = crate::ui_scale::numeric_font(ui);
    let _border_w = ui.push_style_var(StyleVar::FrameBorderSize(1.0));
    let _text_c = invalid.then(|| ui.push_style_color(StyleColor::Text, ERROR_RED));
    let _border_c = invalid.then(|| ui.push_style_color(StyleColor::Border, ERROR_RED));
    let _ = ui.input_text(id, value).build();
}

fn draw_angle_row(ui: &Ui, label: &str, id: &str, value: &mut String) {
    let invalid = parse_angle_deg(value).is_none_or(|a| a <= 0.0 || a >= 90.0);
    crate::theme::property_label(ui, label);
    let _numeric = crate::ui_scale::numeric_font(ui);
    let _border_w = ui.push_style_var(StyleVar::FrameBorderSize(1.0));
    let _text_c = invalid.then(|| ui.push_style_color(StyleColor::Text, ERROR_RED));
    let _border_c = invalid.then(|| ui.push_style_color(StyleColor::Border, ERROR_RED));
    let _ = ui.input_text(id, value).build();
}

fn draw_pair_toggle(
    ui: &Ui,
    first_label: &str,
    second_label: &str,
    id: &str,
    second_selected: &mut bool,
) {
    let _b1 = (!*second_selected)
        .then(|| ui.push_style_color(StyleColor::Button, [0.24, 0.45, 0.78, 1.0]));
    if ui.button(format!("{first_label}{id}-1")) {
        *second_selected = false;
    }
    drop(_b1);
    ui.same_line_with_spacing(0.0, 4.0);
    let _b2 = (*second_selected)
        .then(|| ui.push_style_color(StyleColor::Button, [0.24, 0.45, 0.78, 1.0]));
    if ui.button(format!("{second_label}{id}-2")) {
        *second_selected = true;
    }
}

pub(super) fn draw_failure_modal(v2: &mut ChamferV2State, ui: &Ui) {
    let Some(message) = v2.failure_modal.clone() else {
        return;
    };
    let mut keep_open = true;
    let mut dismiss = false;
    ui.window("Create Chamfer")
        .opened(&mut keep_open)
        .always_auto_resize(true)
        .collapsible(false)
        .build(|| {
            ui.text_colored(ERROR_RED, "Creating the chamfer feature failed.");
            if !message.is_empty() {
                ui.text_wrapped(&message);
            }
            ui.separator();
            if ui.button("Edit") {
                dismiss = true;
            }
            ui.same_line_with_spacing(0.0, 6.0);
            if ui.button("Cancel##chamfer-fail-cancel") {
                dismiss = true;
            }
        });
    if dismiss || !keep_open {
        v2.failure_modal = None;
    }
}
