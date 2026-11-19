// SPDX-License-Identifier: GPL-2.0-only
//! Parameter header and cell context menus stage table changes (#4129).
use imgui::Ui;
use oblikovati_api::wire::ModelStateTableView;

#[derive(Debug, PartialEq)]
pub(super) enum GridAction {
    InsertRow(usize),
    DeleteRow(usize),
    DeleteParameter(u64),
}

pub(super) fn parameter_header(ui: &Ui, parameter: u64) -> Option<GridAction> {
    let _menu = open_context(ui, "##parameter-header-menu")?;
    let action = ui
        .menu_item("Delete Column")
        .then_some(GridAction::DeleteParameter(parameter));
    ui.separator();
    disabled(
        ui,
        &[
            "Key",
            "Custom Parameter Column",
            "Specify Range for Column…",
            "Specify Increment for Column…",
        ],
    );
    ui.separator();
    disabled(ui, &["File Name Column", "Appearance / Material Column"]);
    action
}

pub(super) fn row(
    ui: &Ui,
    view: &ModelStateTableView,
    index: usize,
    parameter: Option<u64>,
) -> Option<GridAction> {
    let _menu = open_context(ui, "##parameter-row-menu")?;
    let mut action = ui
        .menu_item("Insert Row")
        .then_some(GridAction::InsertRow(index));
    let deletable = crate::model_state_table_rows::can_delete(view, index);
    if ui.menu_item_config("Delete Row").enabled(deletable).build() {
        action = Some(GridAction::DeleteRow(index));
    }
    ui.separator();
    disabled(
        ui,
        &[
            "Set As Default / Compare Base Row",
            "Custom Parameter Column",
            "Column Range / Increment",
            "Custom Parameter Cell",
            "Cell Range / Increment",
        ],
    );
    ui.separator();
    if ui
        .menu_item_config("Delete Column")
        .enabled(parameter.is_some())
        .build()
    {
        action = parameter.map(GridAction::DeleteParameter);
    }
    action
}

fn disabled(ui: &Ui, labels: &[&str]) {
    for label in labels {
        ui.menu_item_config(label).enabled(false).build();
    }
}

fn open_context<'ui>(ui: &'ui Ui, popup: &str) -> Option<imgui::PopupToken<'ui>> {
    if ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Right) {
        ui.open_popup(popup);
    }
    ui.begin_popup(popup)
}
