// SPDX-License-Identifier: GPL-2.0-only
//! Mode palette, selection mode bar, constant/variable/setback controls, Advanced Properties, and partial-blend modal.
use super::types::{
    GuiFilletCornerSetback, GuiFilletSelectionMode, GuiFilletV2Action, GuiFilletVariablePoint,
};
use super::{FilletV2State, parse_positive_length};
use crate::feature_properties::draw_pick_pill;
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::fillet_options::{FilletContinuity, FilletMode};

pub(super) fn draw_fillet_body(
    v2: &mut FilletV2State,
    ui: &Ui,
    state: &GuiCommandState,
    radius: &mut String,
    is_edit: bool,
    edited_edges: usize,
    actions: &mut Vec<GuiShellAction>,
) {
    let error = v2.local_error(radius);
    if let Some(msg) = error {
        ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
    }

    draw_mode_palette(v2, ui, state, actions);
    if v2.mode == FilletMode::Constant {
        draw_selection_mode_bar(v2, ui, actions);
    }
    ui.separator();

    match v2.mode {
        FilletMode::Constant => draw_constant_sets(
            v2,
            ui,
            state,
            radius,
            is_edit,
            edited_edges,
            error.is_some(),
        ),
        FilletMode::Variable => {
            draw_variable_controls(v2, ui, state, radius, is_edit, edited_edges)
        }
        FilletMode::CornerSetback => draw_corner_setbacks(v2, ui, state, radius),
    }

    ui.separator();
    draw_advanced_properties(v2, ui);
}

fn draw_mode_palette(
    v2: &mut FilletV2State,
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    let _pad = ui.push_style_var(imgui::StyleVar::FramePadding([2.0, 2.0]));
    let row_w = (ui.content_region_avail()[0] - 6.0).max(160.0);
    for (idx, (mode, label, share)) in [
        (FilletMode::Constant, "Constant", 0.33_f32),
        (FilletMode::Variable, "Variable", 0.33),
        (FilletMode::CornerSetback, "Setbacks", 0.34),
    ]
    .into_iter()
    .enumerate()
    {
        if idx > 0 {
            ui.same_line_with_spacing(0.0, 3.0);
        }
        if toggle_button(ui, label, v2.mode == mode, [row_w * share, 22.0]) && v2.mode != mode {
            v2.mode = mode;
            v2.used_v2 = true;
            match mode {
                FilletMode::CornerSetback => {
                    v2.selection_mode = GuiFilletSelectionMode::CornerVertex;
                    actions.push(GuiShellAction::FilletV2(
                        GuiFilletV2Action::SetSelectionMode(GuiFilletSelectionMode::CornerVertex),
                    ));
                    let has_vertex = state
                        .fillet_v2_preview
                        .as_ref()
                        .is_some_and(|p| !p.available_vertices.is_empty());
                    v2.status_notice = if has_vertex {
                        None
                    } else {
                        Some("No vertex available".into())
                    };
                }
                FilletMode::Variable => {
                    v2.status_notice = None;
                    v2.selection_mode = GuiFilletSelectionMode::Edge;
                    actions.push(GuiShellAction::FilletV2(
                        GuiFilletV2Action::SetSelectionMode(GuiFilletSelectionMode::Edge),
                    ));
                }
                FilletMode::Constant => {
                    v2.status_notice = None;
                    v2.selection_mode = GuiFilletSelectionMode::Edge;
                    actions.push(GuiShellAction::FilletV2(
                        GuiFilletV2Action::SetSelectionMode(GuiFilletSelectionMode::Edge),
                    ));
                }
            }
        }
    }
}

fn draw_selection_mode_bar(v2: &mut FilletV2State, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
    let _pad = ui.push_style_var(imgui::StyleVar::FramePadding([2.0, 2.0]));
    let row_w = (ui.content_region_avail()[0] - 9.0).max(160.0);
    for (idx, (mode, label, share)) in [
        (GuiFilletSelectionMode::Edge, "Edge", 0.22_f32),
        (GuiFilletSelectionMode::Loop, "Loop", 0.22),
        (GuiFilletSelectionMode::Feature, "Feature", 0.30),
        (GuiFilletSelectionMode::Solid, "Solid", 0.26),
    ]
    .into_iter()
    .enumerate()
    {
        if idx > 0 {
            ui.same_line_with_spacing(0.0, 3.0);
        }
        if toggle_button(ui, label, v2.selection_mode == mode, [row_w * share, 20.0])
            && v2.selection_mode != mode
        {
            v2.selection_mode = mode;
            v2.used_v2 = true;
            actions.push(GuiShellAction::FilletV2(
                GuiFilletV2Action::SetSelectionMode(mode),
            ));
        }
    }
}

fn draw_constant_sets(
    v2: &mut FilletV2State,
    ui: &Ui,
    state: &GuiCommandState,
    radius: &mut String,
    is_edit: bool,
    edited_edges: usize,
    invalid: bool,
) {
    if v2.solid_selected {
        crate::theme::property_label(ui, "Solid");
        draw_pick_pill(ui, "1 Solid", true);
    }
    if v2.edge_sets.len() > 1 {
        for idx in 0..v2.edge_sets.len() {
            let _id = ui.push_id_usize(idx);
            let count = v2.edge_sets[idx].edges.len();
            let pill = format_edge_count(count, v2.edge_sets[idx].is_loop);
            crate::theme::property_label(ui, &format!("Set {}", idx + 1));
            draw_pick_pill(ui, &pill, count > 0);
            let set_bad = parse_positive_length(&v2.edge_sets[idx].radius).is_none();
            numeric_input(
                ui,
                "Radius",
                "##set-r",
                &mut v2.edge_sets[idx].radius,
                set_bad,
            );
        }
        if let Some(first) = v2.edge_sets.first() {
            *radius = first.radius.clone();
        }
        return;
    }

    let (count, is_loop) = if let Some(set) = v2.edge_sets.first()
        && !set.edges.is_empty()
    {
        (set.edges.len(), set.is_loop)
    } else if is_edit {
        (edited_edges, false)
    } else {
        (state.selected_edge_count, false)
    };
    let selection = format_edge_count(count, is_loop);
    crate::theme::property_label(ui, "Edges");
    draw_pick_pill(ui, &selection, count > 0);
    if is_loop {
        ui.same_line_with_spacing(0.0, 4.0);
        ui.text_colored([0.46, 0.78, 0.96, 1.0], "[Loop]");
    }
    numeric_input(ui, "Radius", "##blend-value", radius, invalid);
    if let Some(first) = v2.edge_sets.first_mut() {
        first.radius = radius.clone();
    }
}

fn draw_variable_controls(
    v2: &mut FilletV2State,
    ui: &Ui,
    state: &GuiCommandState,
    radius: &mut String,
    is_edit: bool,
    edited_edges: usize,
) {
    ui.text("Variable Radius Fillets");
    let count = if let Some(set) = v2.edge_sets.first()
        && !set.edges.is_empty()
    {
        set.edges.len()
    } else if is_edit {
        edited_edges
    } else {
        state.selected_edge_count
    };
    let summary = variable_edge_summary(count, &v2.variable_points);
    crate::theme::property_label(ui, "Edges");
    draw_pick_pill(ui, &summary, count > 0);

    ui.checkbox("Smooth Radius Transition", &mut v2.smooth_radius_transition);

    let cont_label = match v2.continuity {
        FilletContinuity::Tangent => "Tangent",
        FilletContinuity::SmoothG2 => "Smooth G2",
    };
    crate::theme::property_label(ui, "Continuity");
    if let Some(_combo) = ui.begin_combo("##fillet-continuity", cont_label) {
        if widget_abi::selectable(
            ui,
            "Tangent",
            v2.continuity == FilletContinuity::Tangent,
            false,
        ) {
            v2.continuity = FilletContinuity::Tangent;
            v2.used_v2 = true;
        }
        if widget_abi::selectable(
            ui,
            "Smooth G2",
            v2.continuity == FilletContinuity::SmoothG2,
            false,
        ) {
            v2.continuity = FilletContinuity::SmoothG2;
            v2.used_v2 = true;
        }
    }

    for idx in 0..v2.variable_points.len() {
        let _id = ui.push_id_usize(idx);
        let pos_label = format!("Pos {}", v2.variable_points[idx].position);
        let bad = parse_positive_length(&v2.variable_points[idx].radius).is_none();
        numeric_input(
            ui,
            &pos_label,
            "##var-r",
            &mut v2.variable_points[idx].radius,
            bad,
        );
    }
    if let Some(first) = v2.variable_points.first() {
        *radius = first.radius.clone();
    }
    if widget_abi::button(ui, "Add Position", [0.0, 20.0]) {
        v2.used_v2 = true;
        let next_pos = if v2.variable_points.iter().any(|p| p.position == "0.5") {
            "1.0"
        } else {
            "0.5"
        };
        v2.variable_points.push(GuiFilletVariablePoint {
            position: next_pos.into(),
            radius: radius.clone(),
        });
    }
}

fn draw_corner_setbacks(v2: &mut FilletV2State, ui: &Ui, state: &GuiCommandState, radius: &str) {
    ui.text("Corner Setbacks");
    let available = state
        .fillet_v2_preview
        .as_ref()
        .map(|p| p.available_vertices.as_slice())
        .unwrap_or(&[]);
    if available.is_empty() && v2.corner_setback.is_none() {
        v2.status_notice = Some("No vertex available".into());
    } else {
        v2.status_notice = None;
    }
    let has_vertex = v2.corner_setback.is_some();
    crate::theme::property_label(ui, "Vertex");
    draw_pick_pill(
        ui,
        if has_vertex {
            "1 Vertex"
        } else {
            "Select Vertex"
        },
        has_vertex,
    );
    if !has_vertex
        && let Some(first) = available.first()
        && widget_abi::button(ui, "Select Shared Vertex", [0.0, 20.0])
    {
        let count = first.incident_edges.len().max(3);
        v2.corner_setback = Some(GuiFilletCornerSetback {
            vertex: first.key.clone(),
            minimal: false,
            distances: vec![radius.into(); count],
        });
        v2.used_v2 = true;
    }
    if let Some(sb) = &mut v2.corner_setback {
        ui.text("Corner Setback Behavior");
        ui.checkbox("Minimal", &mut sb.minimal);
        for idx in 0..sb.distances.len() {
            let _id = ui.push_id_usize(idx);
            let label = format!("Edge {}", idx + 1);
            let bad = !sb.minimal && parse_positive_length(&sb.distances[idx]).is_none();
            let _disabled = ui.begin_disabled(sb.minimal);
            numeric_input(ui, &label, "##sb-d", &mut sb.distances[idx], bad);
        }
    }
}

fn draw_advanced_properties(v2: &mut FilletV2State, ui: &Ui) {
    let header = if v2.advanced_open {
        "v Advanced Properties"
    } else {
        "> Advanced Properties"
    };
    if widget_abi::selectable(ui, header, false, false) {
        v2.advanced_open = !v2.advanced_open;
    }
    if !v2.advanced_open {
        return;
    }
    ui.checkbox(
        "Roll Along Sharp Edges",
        &mut v2.advanced.roll_along_sharp_edges,
    );
    ui.checkbox(
        "Rolling Ball Where Possible",
        &mut v2.advanced.rolling_ball_where_possible,
    );
    ui.checkbox(
        "Preserve All Features",
        &mut v2.advanced.preserve_all_features,
    );
    let _disabled = ui.begin_disabled(true);
    ui.checkbox(
        "Automatic Edge Chain",
        &mut v2.advanced.automatic_edge_chain,
    );
}

pub(super) fn draw_partial_modal(
    v2: &mut FilletV2State,
    ui: &Ui,
    _actions: &mut Vec<GuiShellAction>,
) -> bool {
    let Some(mut modal) = v2.partial_modal.take() else {
        return false;
    };
    let mut keep_open = true;
    let mut accept_now = false;
    ui.window("Create Fillet")
        .always_auto_resize(true)
        .collapsible(false)
        .build(|| {
            ui.text(format!(
                "{} of {} edges cannot be blended at the current size.",
                modal.failed_count, modal.total_count
            ));
            ui.checkbox(
                "Accept successful blends and change body selection to edge selection",
                &mut modal.accept_successful,
            );
            ui.separator();
            let _disabled = ui.begin_disabled(!modal.accept_successful);
            if widget_abi::button(ui, "OK", [72.0, 0.0]) {
                accept_now = true;
                keep_open = false;
            }
            drop(_disabled);
            ui.same_line_with_spacing(0.0, 6.0);
            if widget_abi::button(ui, "Cancel", [72.0, 0.0]) {
                keep_open = false;
            }
        });
    if accept_now {
        v2.solid_selected = false;
        v2.selection_mode = GuiFilletSelectionMode::Edge;
        v2.edge_sets = modal.kept_sets;
        v2.used_v2 = true;
        v2.pending_commit_after_partial = true;
        return true;
    }
    if keep_open {
        v2.partial_modal = Some(modal);
    }
    false
}

fn format_edge_count(count: usize, _is_loop: bool) -> String {
    match count {
        0 => "Select Edges".to_owned(),
        1 => "1 Edge".to_owned(),
        n => format!("{n} Edges"),
    }
}

fn variable_edge_summary(count: usize, pts: &[GuiFilletVariablePoint]) -> String {
    let base = format_edge_count(count, false);
    if count == 0 || pts.len() < 2 {
        return base;
    }
    let mut vals: Vec<f64> = pts
        .iter()
        .filter_map(|p| parse_positive_length(&p.radius))
        .collect();
    if vals.len() < 2 {
        return base;
    }
    vals.sort_by(f64::total_cmp);
    let lo = format_mm_short(vals[0]);
    let hi = format_mm_short(*vals.last().unwrap());
    format!("{base} ({lo} to {hi} mm)")
}

fn format_mm_short(v: f64) -> String {
    if v.fract().abs() < 1e-6 {
        format!("{}", v as i64)
    } else {
        format!("{v:.1}")
    }
}

fn toggle_button(ui: &Ui, label: &str, selected: bool, size: [f32; 2]) -> bool {
    let color = selected.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Button, [0.26, 0.36, 0.68, 1.0])
    });
    let clicked = widget_abi::button(ui, label, size);
    drop(color);
    clicked
}

fn numeric_input(ui: &Ui, label: &str, id: &str, value: &mut String, invalid: bool) -> bool {
    crate::theme::property_label(ui, label);
    let _numeric = crate::ui_scale::numeric_font(ui);
    let _outline = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.0));
    let _border = invalid.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Border, [0.88, 0.25, 0.25, 1.0])
    });
    ui.input_text(id, value).build()
}
