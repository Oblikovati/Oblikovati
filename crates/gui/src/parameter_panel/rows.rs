// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn draw_parameter_row(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    drafts: &mut GuiParameterDrafts,
    actions: &mut Vec<GuiParameterAction>,
) {
    let _identity = ui.push_id(format!("document:{document}:parameter:{}", row.parameter));
    let editable = row.is_editable();
    ui.table_next_row();
    text_cell(
        ui,
        document,
        row,
        GuiParameterTextField::Name,
        editable,
        drafts,
        actions,
    );
    ui.table_next_column();
    ui.text(&row.detail.units);
    equation_cell(ui, document, row, editable, drafts, actions);
    ui.table_next_column();
    draw_parameter_value(ui, row);
    ui.table_next_column();
    draw_parameter_tolerance(ui, row);
    toggle_cell(
        ui,
        document,
        row,
        GuiParameterToggleField::Key,
        row.detail.is_key,
        editable,
        actions,
    );
    toggle_cell(
        ui,
        document,
        row,
        GuiParameterToggleField::Export,
        row.detail.exposed_as_property,
        editable,
        actions,
    );
    text_cell(
        ui,
        document,
        row,
        GuiParameterTextField::Comment,
        editable,
        drafts,
        actions,
    );
}
fn text_cell(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    field: GuiParameterTextField,
    editable: bool,
    drafts: &mut GuiParameterDrafts,
    actions: &mut Vec<GuiParameterAction>,
) {
    ui.table_next_column();
    let (label, seed) = parameter_text_seed(row, field);
    if !editable {
        ui.text(seed);
        if field == GuiParameterTextField::Name {
            draw_parameter_row_menu(ui, document, row, false, actions);
        }
        return;
    }
    let target = GuiParameterTextTarget {
        document,
        parameter: row.parameter,
        field,
    };
    draw_text_editor(ui, (label, seed), target, Some(row), drafts, actions);
}
fn draw_text_editor(
    ui: &Ui,
    text: (&str, &str),
    target: GuiParameterTextTarget,
    row: Option<&GuiParameterRow>,
    drafts: &mut GuiParameterDrafts,
    actions: &mut Vec<GuiParameterAction>,
) {
    let (label, seed) = text;
    ui.set_next_item_width(-1.);
    if drafts.is_pending(target) {
        let mut text = drafts.text(target).unwrap_or(seed).to_owned();
        let _disabled = ui.begin_disabled(true);
        ui.input_text(label, &mut text).build();
        return;
    }
    let outcome = edit_text_widget(ui, label, seed, target, drafts);
    if target.field == GuiParameterTextField::Name
        && let Some(row) = row
    {
        draw_parameter_row_menu(ui, target.document, row, true, actions);
    }
    match outcome {
        Ok(Some(edit)) => actions.push(GuiParameterAction::Text(edit)),
        Ok(None) => {}
        Err(issue) => ui.text_wrapped(issue.replace('\0', "\\0")),
    }
    draw_parameter_draft_issue(ui, drafts, target);
}
fn edit_text_widget(
    ui: &Ui,
    label: &str,
    seed: &str,
    target: GuiParameterTextTarget,
    drafts: &mut GuiParameterDrafts,
) -> Result<Option<GuiParameterTextEdit>, String> {
    ui.input_text(label, drafts.text_buffer(target, seed)?)
        .build();
    if !ui.is_item_deactivated_after_edit() {
        return Ok(None);
    }
    drafts.submit(target)
}
fn parameter_text_seed(
    row: &GuiParameterRow,
    field: GuiParameterTextField,
) -> (&'static str, &str) {
    match field {
        GuiParameterTextField::Name => ("##name", &row.detail.parameter_info.name),
        GuiParameterTextField::Equation => ("##equation", &row.detail.parameter_info.expression),
        GuiParameterTextField::Comment => ("##comment", &row.detail.comment),
    }
}
fn equation_cell(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    editable: bool,
    drafts: &mut GuiParameterDrafts,
    actions: &mut Vec<GuiParameterAction>,
) {
    if row.detail.expression_list.is_some() {
        draw_equation_choices(ui, document, row, editable, drafts, actions);
        return;
    }
    if row.detail.units == "boolean" {
        toggle_cell(
            ui,
            document,
            row,
            GuiParameterToggleField::Boolean,
            row.detail.parameter_info.expression == "true",
            editable,
            actions,
        );
        return;
    }
    text_cell(
        ui,
        document,
        row,
        GuiParameterTextField::Equation,
        editable,
        drafts,
        actions,
    );
}
fn draw_equation_choices(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    editable: bool,
    drafts: &mut GuiParameterDrafts,
    actions: &mut Vec<GuiParameterAction>,
) {
    ui.table_next_column();
    ui.set_next_item_width(-1.);
    let target = GuiParameterTextTarget {
        document,
        parameter: row.parameter,
        field: GuiParameterTextField::Equation,
    };
    draw_parameter_choice_combo(ui, row, target, editable, drafts, actions);
    draw_parameter_draft_issue(ui, drafts, target);
}
fn draw_parameter_choice_combo(
    ui: &Ui,
    row: &GuiParameterRow,
    target: GuiParameterTextTarget,
    editable: bool,
    drafts: &mut GuiParameterDrafts,
    actions: &mut Vec<GuiParameterAction>,
) {
    let _disabled = ui.begin_disabled(!editable || drafts.is_pending(target));
    let Some(_combo) = ui.begin_combo("##equation-list", &row.detail.parameter_info.expression)
    else {
        return;
    };
    for choice in choice_labels(row) {
        if !ui.selectable(choice) {
            continue;
        }
        match choose_parameter_expression(
            drafts,
            target,
            &row.detail.parameter_info.expression,
            choice,
        ) {
            Ok(Some(edit)) => actions.push(GuiParameterAction::Text(edit)),
            Ok(None) => {}
            Err(issue) => ui.text_wrapped(issue.replace('\0', "\\0")),
        }
    }
}
fn choice_labels(row: &GuiParameterRow) -> Vec<&str> {
    let choices = row
        .detail
        .expression_list
        .as_deref()
        .and_then(|list| list.expressions.as_deref())
        .unwrap_or_default();
    let current = row.detail.parameter_info.expression.as_str();
    let mut labels = choices.iter().map(String::as_str).collect::<Vec<_>>();
    if !labels.contains(&current) {
        labels.push(current);
    }
    labels
}
fn draw_parameter_draft_issue(
    ui: &Ui,
    drafts: &GuiParameterDrafts,
    target: GuiParameterTextTarget,
) {
    if let Some(issue) = drafts.issue(target) {
        ui.text_wrapped(issue.replace('\0', "\\0"));
    }
}
fn choose_parameter_expression(
    drafts: &mut GuiParameterDrafts,
    target: GuiParameterTextTarget,
    seed: &str,
    choice: &str,
) -> Result<Option<GuiParameterTextEdit>, String> {
    *drafts.text_buffer(target, seed)? = choice.into();
    drafts.submit(target)
}
fn toggle_cell(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    field: GuiParameterToggleField,
    mut value: bool,
    editable: bool,
    actions: &mut Vec<GuiParameterAction>,
) {
    ui.table_next_column();
    let _disabled = ui.begin_disabled(!editable);
    let label = match field {
        GuiParameterToggleField::Boolean => "##boolean",
        GuiParameterToggleField::Key => "##key",
        GuiParameterToggleField::Export => "##export",
    };
    if ui.checkbox(label, &mut value) {
        actions.push(GuiParameterAction::Toggle {
            document,
            parameter: row.parameter,
            field,
            value,
        });
    }
}
fn draw_parameter_value(ui: &Ui, row: &GuiParameterRow) {
    ui.text(parameter_value_label(row));
    if !row.detail.parameter_info.health.is_empty() && ui.is_item_hovered() {
        ui.tooltip_text(&row.detail.parameter_info.health);
    }
}
fn parameter_value_label(row: &GuiParameterRow) -> String {
    let info = &row.detail.parameter_info;
    if info.health.is_empty() {
        return info.value.clone();
    }
    format!("! {}", info.value)
}
fn draw_parameter_tolerance(ui: &Ui, row: &GuiParameterRow) {
    ui.text(&row.tolerance_text);
}
#[cfg(test)]
#[path = "../../tests/parameters/cells.rs"]
mod tests;

fn draw_parameter_row_menu(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    editable: bool,
    actions: &mut Vec<GuiParameterAction>,
) {
    let Some(_popup) = ui.begin_popup_context_with_label("##parameter-row-menu") else {
        return;
    };
    draw_parameter_row_menu_items(ui, document, row.parameter, editable, actions);
    draw_parameter_value_menu(ui, document, row, editable, actions);
    draw_parameter_tolerance_menu(ui, document, row, editable, actions);
    draw_parameter_group_menu(ui, document, row, actions);
}
fn draw_parameter_row_menu_items(
    ui: &Ui,
    document: u64,
    parameter: u64,
    editable: bool,
    actions: &mut Vec<GuiParameterAction>,
) {
    for (label, operation, enabled) in [
        (
            "Copy to User Parameter",
            GuiParameterRowOperation::CopyToUser,
            true,
        ),
        (
            "Delete Parameter",
            GuiParameterRowOperation::Delete,
            editable,
        ),
    ] {
        if ui.menu_item_config(label).enabled(enabled).build() {
            actions.push(GuiParameterAction::Row {
                document,
                parameter,
                operation,
            });
        }
    }
}

fn draw_parameter_value_menu(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    editable: bool,
    actions: &mut Vec<GuiParameterAction>,
) {
    if !editable || row.detail.units == "boolean" {
        return;
    }
    let has_list = row.detail.expression_list.is_some();
    let label = if has_list {
        "Edit Multi-Value List…"
    } else {
        "Make Multi-Value…"
    };
    if ui.menu_item(label) {
        actions.push(GuiParameterAction::Row {
            document,
            parameter: row.parameter,
            operation: GuiParameterRowOperation::EditValueList,
        });
    }
    if has_list && ui.menu_item("Make Single Value") {
        actions.push(GuiParameterAction::Row {
            document,
            parameter: row.parameter,
            operation: GuiParameterRowOperation::ClearValueList,
        });
    }
}

fn draw_parameter_tolerance_menu(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    editable: bool,
    actions: &mut Vec<GuiParameterAction>,
) {
    if !editable || matches!(row.detail.units.as_str(), "text" | "boolean") {
        return;
    }
    if ui.menu_item("Edit Tolerance…") {
        actions.push(GuiParameterAction::Row {
            document,
            parameter: row.parameter,
            operation: GuiParameterRowOperation::EditTolerance,
        });
    }
}

fn draw_parameter_group_menu(
    ui: &Ui,
    document: u64,
    row: &GuiParameterRow,
    actions: &mut Vec<GuiParameterAction>,
) {
    if ui.menu_item("Add to Group…") {
        actions.push(GuiParameterAction::Row {
            document,
            parameter: row.parameter,
            operation: GuiParameterRowOperation::AddToGroup,
        });
    }
    if row.has_groups && ui.menu_item("Remove from Group") {
        actions.push(GuiParameterAction::Row {
            document,
            parameter: row.parameter,
            operation: GuiParameterRowOperation::RemoveFromGroups,
        });
    }
}
