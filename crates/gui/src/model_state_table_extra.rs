// SPDX-License-Identifier: GPL-2.0-only
//! Edit suppression and parameter columns in the shared table draft (#4119).
use imgui::{TableFlags, Ui};
use oblikovati_api::{
    types::{MODEL_STATE_TABLE_PARAMETER_COLUMN, MODEL_STATE_TABLE_SUPPRESSION_COLUMN},
    wire::*,
};
use std::collections::BTreeMap;

#[derive(Default)]
pub(crate) struct ExtraSelection {
    feature_available: Option<u64>,
    feature_chosen: Option<u64>,
    parameter_available: Option<u64>,
    parameter_chosen: Option<u64>,
    buffers: BTreeMap<(String, u64), String>,
    pub(crate) diagnostics: Vec<ModelStateTableDiagnostic>,
    pub(crate) verify_requested: bool,
}
#[derive(Clone, Copy)]
pub(crate) enum ExtraKind {
    Suppression,
    Parameters,
}

pub(crate) fn selection(
    ui: &Ui,
    view: &mut ModelStateTableView,
    selected: &mut ExtraSelection,
    kind: ExtraKind,
) -> bool {
    let Some(_table) = crate::widget_abi::begin_table_with_sizing(
        ui,
        "extra-selection",
        3,
        TableFlags::BORDERS | TableFlags::RESIZABLE,
        [0., ui.frame_height_with_spacing() * 8.],
        0.,
    ) else {
        return false;
    };
    ui.table_next_row();
    ui.table_next_column();
    available(ui, view, selected, kind);
    ui.table_next_column();
    let mut changed = false;
    let available = match kind {
        ExtraKind::Suppression => selected.feature_available,
        ExtraKind::Parameters => selected.parameter_available,
    };
    let chosen_id = match kind {
        ExtraKind::Suppression => selected.feature_chosen,
        ExtraKind::Parameters => selected.parameter_chosen,
    };
    let disabled = ui.begin_disabled(available.is_none());
    if crate::widget_abi::button(ui, "Add >", [0.; 2]) {
        changed |= add(view, selected, kind);
    }
    drop(disabled);
    let disabled = ui.begin_disabled(chosen_id.is_none());
    if crate::widget_abi::button(ui, "< Remove", [0.; 2]) {
        changed |= remove(view, selected, kind);
    }
    drop(disabled);
    ui.table_next_column();
    chosen(ui, view, selected, kind);
    changed
}
fn available(ui: &Ui, view: &ModelStateTableView, selected: &mut ExtraSelection, kind: ExtraKind) {
    match kind {
        ExtraKind::Suppression => {
            for item in view.available_features.as_deref().unwrap_or_default() {
                if ui
                    .selectable_config(&item.name)
                    .selected(selected.feature_available == Some(item.id))
                    .build()
                {
                    selected.feature_available = Some(item.id);
                }
            }
        }
        ExtraKind::Parameters => {
            let parameters = view.available_parameters.as_deref().unwrap_or_default();
            for feature in view.available_features.as_deref().unwrap_or_default() {
                if !parameters.iter().any(|item| item.feature_id == feature.id) {
                    continue;
                }
                let _id = ui.push_id(format!("feature-{}", feature.id));
                let Some(_tree) = ui.tree_node_config(&feature.name).default_open(true).push()
                else {
                    continue;
                };
                for item in parameters
                    .iter()
                    .filter(|item| item.feature_id == feature.id)
                {
                    parameter_choice(ui, view, selected, item);
                }
            }
            for item in parameters.iter().filter(|item| item.feature_id == 0) {
                parameter_choice(ui, view, selected, item);
            }
        }
    }
}
fn parameter_choice(
    ui: &Ui,
    view: &ModelStateTableView,
    selected: &mut ExtraSelection,
    item: &ModelStateTableParameterChoice,
) {
    let _id = ui.push_id(format!("parameter-{}", item.id));
    if ui
        .selectable_config(&item.name)
        .selected(selected.parameter_available == Some(item.id))
        .build()
    {
        selected.parameter_available = Some(item.id);
    }
    if let Some(expression) = active_expression(view, item.id) {
        ui.same_line();
        ui.text(format!("= {expression}"));
    }
}
fn active_expression(view: &ModelStateTableView, id: u64) -> Option<&str> {
    if let Some(column) = view
        .parameters
        .as_deref()
        .unwrap_or_default()
        .iter()
        .find(|item| item.parameter_id == id)
    {
        let cell = column
            .cells
            .as_deref()
            .unwrap_or_default()
            .iter()
            .find(|cell| cell.row.member_id == view.active_member_id);
        return Some(
            cell.and_then(|cell| cell.expression.as_deref().map(String::as_str))
                .unwrap_or(&column.base),
        );
    }
    view.available_parameters
        .as_deref()
        .unwrap_or_default()
        .iter()
        .find(|item| item.id == id)
        .map(|item| item.base_expression.as_str())
}
fn chosen(ui: &Ui, view: &ModelStateTableView, selected: &mut ExtraSelection, kind: ExtraKind) {
    ui.text(match kind {
        ExtraKind::Suppression => "Chosen features",
        ExtraKind::Parameters => "Chosen parameters",
    });
    for column in view.columns.as_deref().unwrap_or_default() {
        let id = match kind {
            ExtraKind::Suppression if column.kind == MODEL_STATE_TABLE_SUPPRESSION_COLUMN => {
                column.feature_id
            }
            ExtraKind::Parameters if column.kind == MODEL_STATE_TABLE_PARAMETER_COLUMN => {
                column.parameter_id
            }
            _ => continue,
        };
        let label = name(view, column);
        let current = match kind {
            ExtraKind::Suppression => &mut selected.feature_chosen,
            ExtraKind::Parameters => &mut selected.parameter_chosen,
        };
        if ui
            .selectable_config(label)
            .selected(*current == Some(id))
            .build()
        {
            *current = Some(id);
        }
    }
}
fn add(view: &mut ModelStateTableView, selected: &ExtraSelection, kind: ExtraKind) -> bool {
    let id = match kind {
        ExtraKind::Suppression => selected.feature_available,
        ExtraKind::Parameters => selected.parameter_available,
    };
    let Some(id) = id else {
        return false;
    };
    let column = match kind {
        ExtraKind::Suppression => ModelStateTableColumnRef {
            kind: MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
            feature_id: id,
            ..Default::default()
        },
        ExtraKind::Parameters => ModelStateTableColumnRef {
            kind: MODEL_STATE_TABLE_PARAMETER_COLUMN,
            parameter_id: id,
            ..Default::default()
        },
    };
    if view
        .columns
        .as_deref()
        .unwrap_or_default()
        .contains(&column)
    {
        return false;
    }
    if !ensure_column(view, id, kind) {
        return false;
    }
    view.columns.get_or_insert_with(Vec::new).push(column);
    true
}
fn ensure_column(view: &mut ModelStateTableView, id: u64, kind: ExtraKind) -> bool {
    let rows: Vec<_> = view
        .rows
        .as_deref()
        .unwrap_or_default()
        .iter()
        .map(|row| row.ref_.clone())
        .collect();
    match kind {
        ExtraKind::Suppression => {
            if view
                .suppressions
                .as_deref()
                .unwrap_or_default()
                .iter()
                .any(|item| item.feature_id == id)
            {
                return true;
            }
            let Some(choice) = view
                .available_features
                .as_deref()
                .unwrap_or_default()
                .iter()
                .find(|item| item.id == id)
            else {
                return false;
            };
            let cells = rows
                .into_iter()
                .map(|row| ModelStateTableSuppressionCell {
                    row,
                    suppressed: None,
                })
                .collect();
            view.suppressions
                .get_or_insert_with(Vec::new)
                .push(ModelStateTableSuppression {
                    feature_id: id,
                    base: choice.base_suppressed,
                    cells: Some(cells),
                });
        }
        ExtraKind::Parameters => {
            if view
                .parameters
                .as_deref()
                .unwrap_or_default()
                .iter()
                .any(|item| item.parameter_id == id)
            {
                return true;
            }
            let Some(choice) = view
                .available_parameters
                .as_deref()
                .unwrap_or_default()
                .iter()
                .find(|item| item.id == id)
            else {
                return false;
            };
            let cells = rows
                .into_iter()
                .map(|row| ModelStateTableParameterCell {
                    row,
                    expression: None,
                })
                .collect();
            view.parameters
                .get_or_insert_with(Vec::new)
                .push(ModelStateTableParameter {
                    parameter_id: id,
                    base: choice.base_expression.clone(),
                    cells: Some(cells),
                });
        }
    }
    true
}
fn remove(view: &mut ModelStateTableView, selected: &mut ExtraSelection, kind: ExtraKind) -> bool {
    let (id, expected) = match kind {
        ExtraKind::Suppression => (
            selected.feature_chosen.take(),
            MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
        ),
        ExtraKind::Parameters => (
            selected.parameter_chosen.take(),
            MODEL_STATE_TABLE_PARAMETER_COLUMN,
        ),
    };
    let Some(id) = id else {
        return false;
    };
    let columns = view.columns.get_or_insert_with(Vec::new);
    let before = columns.len();
    columns.retain(|column| {
        column.kind != expected
            || match kind {
                ExtraKind::Suppression => column.feature_id != id,
                ExtraKind::Parameters => column.parameter_id != id,
            }
    });
    before != columns.len()
}
pub(crate) fn remove_parameter(
    view: &mut ModelStateTableView,
    selected: &mut ExtraSelection,
    id: u64,
) -> bool {
    let Some(columns) = view.columns.as_mut() else {
        return false;
    };
    let before = columns.len();
    columns.retain(|column| {
        column.kind != MODEL_STATE_TABLE_PARAMETER_COLUMN || column.parameter_id != id
    });
    if columns.len() == before {
        return false;
    }
    if selected.parameter_chosen == Some(id) {
        selected.parameter_chosen = None;
    }
    true
}
pub(crate) fn name<'a>(
    view: &'a ModelStateTableView,
    column: &ModelStateTableColumnRef,
) -> &'a str {
    if column.kind == MODEL_STATE_TABLE_SUPPRESSION_COLUMN {
        return view
            .available_features
            .as_deref()
            .unwrap_or_default()
            .iter()
            .find(|item| item.id == column.feature_id)
            .map_or("Feature", |item| item.name.as_str());
    }
    view.available_parameters
        .as_deref()
        .unwrap_or_default()
        .iter()
        .find(|item| item.id == column.parameter_id)
        .map_or("Parameter", |item| item.name.as_str())
}
pub(crate) fn cell(
    ui: &Ui,
    view: &mut ModelStateTableView,
    selected: &mut ExtraSelection,
    row: usize,
    column: &ModelStateTableColumnRef,
) -> bool {
    if column.kind == MODEL_STATE_TABLE_SUPPRESSION_COLUMN {
        return suppression_cell(ui, view, row, column.feature_id);
    }
    if column.kind == MODEL_STATE_TABLE_PARAMETER_COLUMN {
        return parameter_cell(ui, view, selected, row, column.parameter_id);
    }
    false
}
fn suppression_cell(ui: &Ui, view: &mut ModelStateTableView, row: usize, id: u64) -> bool {
    let Some(column) = view
        .suppressions
        .as_mut()
        .and_then(|items| items.iter_mut().find(|item| item.feature_id == id))
    else {
        return false;
    };
    let Some(cell) = column.cells.as_mut().and_then(|cells| cells.get_mut(row)) else {
        return false;
    };
    let suppressed = cell.suppressed.as_deref().copied().unwrap_or(column.base);
    let row = if cell.row.member_id != 0 {
        format!("member-{}", cell.row.member_id)
    } else {
        format!("draft-{}", cell.row.draft_key)
    };
    let label = format!("##suppress-{row}-{id}");
    let Some(_combo) = ui.begin_combo(label, if suppressed { "Suppress" } else { "Compute" })
    else {
        return false;
    };
    let mut changed = false;
    for (title, value) in [("Compute", false), ("Suppress", true)] {
        if ui
            .selectable_config(title)
            .selected(suppressed == value)
            .build()
            && suppressed != value
        {
            cell.suppressed = Some(Box::new(value));
            changed = true;
        }
    }
    changed
}
fn parameter_cell(
    ui: &Ui,
    view: &mut ModelStateTableView,
    selected: &mut ExtraSelection,
    row: usize,
    id: u64,
) -> bool {
    let Some(column) = view
        .parameters
        .as_mut()
        .and_then(|items| items.iter_mut().find(|item| item.parameter_id == id))
    else {
        return false;
    };
    let Some(cell) = column.cells.as_mut().and_then(|cells| cells.get_mut(row)) else {
        return false;
    };
    let row = if cell.row.member_id != 0 {
        format!("member-{}", cell.row.member_id)
    } else {
        format!("draft-{}", cell.row.draft_key)
    };
    let key = (row, id);
    let label = format!("##parameter-{}-{id}", key.0);
    let text = selected.buffers.entry(key).or_insert_with(|| {
        cell.expression
            .as_deref()
            .cloned()
            .unwrap_or_else(|| column.base.clone())
    });
    let invalid = selected.diagnostics.iter().any(|diagnostic| {
        diagnostic.parameter_id == id && diagnostic.row.as_deref() == Some(&cell.row)
    });
    let color =
        invalid.then(|| ui.push_style_color(imgui::StyleColor::FrameBg, [0.75, 0.58, 0.12, 1.0]));
    let changed = ui.input_text(label, text).build();
    if (ui.is_item_active() || ui.is_item_deactivated_after_edit())
        && ui.is_key_pressed(imgui::Key::Enter)
    {
        selected.verify_requested = true;
    }
    drop(color);
    if changed {
        cell.expression = Some(Box::new(text.clone()));
    }
    changed
}

#[cfg(test)]
#[path = "model_state_table_extra_tests.rs"]
mod tests;
