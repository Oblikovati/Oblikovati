// SPDX-License-Identifier: GPL-2.0-only
//! Child operation rows (e.g. `Scale1`) rendered under a Direct Edit node in the Model Browser (#3918).
use crate::{
    GuiBrowserFeature, GuiCommandState, GuiDirectEditAction, GuiModelBrowser, GuiShellAction,
};
use imgui::{MouseButton, Ui};

pub(crate) fn direct_edit_operation_rows(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(seed) = browser.direct_edit_edits.get(&feature.id) else {
        return;
    };
    for (index, op) in seed.operations.iter().enumerate() {
        let _id = ui.push_id(format!("direct-edit-op-{index}"));
        let clicked = ui
            .selectable_config(format!("  {}###de-op-{}-{}", op.name, feature.id.0, index))
            .build();
        let open_action = GuiShellAction::DirectEdit(GuiDirectEditAction::OpenEdit {
            feature: feature.id,
            operation: Some(index),
        });
        let hovered = ui.is_item_hovered();
        if (clicked || (hovered && ui.is_mouse_double_clicked(MouseButton::Left)))
            && open_action.enabled(state)
        {
            actions.push(open_action.clone());
        }
        if let Some(_menu) = ui.begin_popup_context_item()
            && ui
                .menu_item_config("Edit Feature")
                .enabled(open_action.enabled(state))
                .build()
        {
            actions.push(open_action);
        }
    }
}
