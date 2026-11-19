// SPDX-License-Identifier: GPL-2.0-only
//! Draw the model-state table and stage its row and column actions (#4129).
use super::*;

impl ModelStateTable {
    pub(super) fn grid(&mut self, ui: &Ui) {
        let Some(view) = self.draft.as_mut() else {
            return;
        };
        let rows = view.rows.clone().unwrap_or_default();
        let columns = view.columns.clone().unwrap_or_default();
        let Some(_table) = ui.begin_table_with_sizing(
            "model-state-grid",
            columns.len() + 2,
            TableFlags::BORDERS | TableFlags::ROW_BG | TableFlags::RESIZABLE,
            [0., 180.],
            0.,
        ) else {
            return;
        };
        ui.table_setup_column("#");
        ui.table_setup_column("Member");
        for column in &columns {
            let name = column.property.as_deref().map_or_else(
                || crate::model_state_table_extra::name(view, column),
                |key| key.name.as_str(),
            );
            ui.table_setup_column(name);
        }
        let mut row_action = headers(ui, view, &columns);
        for (index, row) in rows.iter().enumerate() {
            ui.table_next_row();
            if row.ref_.member_id != 0 && row.ref_.member_id == view.active_member_id {
                // #4130: show which committed member supplies the tree values.
                ui.table_set_bg_color(
                    TableBgTarget::ROW_BG0,
                    ImColor32::from_rgba(184, 224, 184, 255),
                );
            }
            ui.table_next_column();
            let _id = ui.push_id(row_key(&row.ref_));
            ui.small_button(format!("{}##row", index + 1));
            if let Some(action) = menus::row(ui, view, index, None) {
                row_action = Some(action);
            }
            ui.table_next_column();
            let mut name = row.name.clone();
            let invalid_name = names::invalid(&rows, index);
            let highlight = invalid_name
                .then(|| ui.push_style_color(imgui::StyleColor::FrameBg, [0.75, 0.58, 0.12, 1.0]));
            let disabled = ui.begin_disabled(index == 0);
            ui.set_next_item_width(140.);
            if ui.input_text("##member-name", &mut name).build()
                && let Some(current) = view.rows.as_mut().and_then(|items| items.get_mut(index))
            {
                current.name = name;
                self.dirty = true;
            }
            drop(disabled);
            drop(highlight);
            for column in &columns {
                ui.table_next_column();
                if let Some(key) = column.property.as_deref() {
                    if let Some(property) = view
                        .properties
                        .as_mut()
                        .and_then(|items| items.iter_mut().find(|item| item.property == *key))
                    {
                        self.dirty |= edit_cell(
                            ui,
                            &row.ref_,
                            index,
                            property,
                            &mut self.buffers,
                            &mut self.invalid,
                        );
                    }
                } else {
                    let _column_id =
                        ui.push_id(format!("column-{:?}-{}", column.kind, column.parameter_id));
                    self.dirty |= crate::model_state_table_extra::cell(
                        ui,
                        view,
                        &mut self.extra,
                        index,
                        column,
                    );
                    if column.kind == oblikovati_api::types::MODEL_STATE_TABLE_PARAMETER_COLUMN
                        && let Some(action) = menus::row(ui, view, index, Some(column.parameter_id))
                    {
                        row_action = Some(action);
                    }
                }
            }
        }
        if let Some(action) = row_action {
            self.dirty |= match action {
                menus::GridAction::InsertRow(index) => {
                    crate::model_state_table_rows::insert(view, index, &mut self.row_sequence)
                }
                menus::GridAction::DeleteRow(index) => {
                    crate::model_state_table_rows::delete(view, index)
                }
                menus::GridAction::DeleteParameter(id) => {
                    crate::model_state_table_extra::remove_parameter(view, &mut self.extra, id)
                }
            };
        }
    }
}

fn headers(
    ui: &Ui,
    view: &ModelStateTableView,
    columns: &[ModelStateTableColumnRef],
) -> Option<menus::GridAction> {
    use oblikovati_api::types::MODEL_STATE_TABLE_PARAMETER_COLUMN;
    ui.table_next_row_with_flags(imgui::TableRowFlags::HEADERS);
    for name in ["#", "Member"] {
        ui.table_next_column();
        ui.table_header(name);
    }
    let mut action = None;
    for column in columns {
        ui.table_next_column();
        let name = column.property.as_deref().map_or_else(
            || crate::model_state_table_extra::name(view, column),
            |key| key.name.as_str(),
        );
        if column.kind != MODEL_STATE_TABLE_PARAMETER_COLUMN {
            ui.table_header(name);
            continue;
        }
        let _id = ui.push_id(format!("parameter-header-{}", column.parameter_id));
        ui.table_header(name);
        if let Some(selected) = menus::parameter_header(ui, column.parameter_id) {
            action = Some(selected);
        }
    }
    action
}

#[path = "model_state_table_grid_menus.rs"]
mod menus;
