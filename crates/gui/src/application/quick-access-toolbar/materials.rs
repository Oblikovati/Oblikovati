// SPDX-License-Identifier: GPL-2.0-only
//! Assign an existing physical material from the quick access toolbar.
use crate::{GuiAuthoringContext, GuiCommandState, GuiPartAssetKind, GuiShellAction};
use imgui::Ui;

const MENU: &str = "##quick-access-materials";

pub(super) fn button(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    open: &mut bool,
) {
    let disabled =
        ui.begin_disabled(state.authoring != GuiAuthoringContext::Part || state.transaction_open);
    let requested = std::mem::take(open);
    if ui.button("Materials") || requested {
        ui.open_popup(MENU);
    }
    drop(disabled);
    if let Some(_menu) = ui.begin_popup(MENU) {
        choices(ui, state, actions);
    }
}

fn choices(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(browser) = &state.browser else {
        return;
    };
    for material in &browser.materials {
        let selected = browser.part_material.as_deref() == Some(material.id.as_str());
        let _id = ui.push_id(&material.id);
        if ui
            .menu_item_config(&material.name)
            .selected(selected)
            .build()
            && !selected
        {
            crate::ribbon_part_assets::queue_asset(
                state,
                GuiPartAssetKind::Material,
                &material.id,
                actions,
            );
        }
    }
}
