// SPDX-License-Identifier: GPL-2.0-only
//! Definition rows and their placement command in the model browser.
use super::{BrowserTree, GuiShellAction, StyleColor};

impl BrowserTree<'_> {
    pub(super) fn blocks(&self, actions: &mut Vec<GuiShellAction>) {
        if self.browser.block_definitions.is_empty() {
            return;
        }
        if let Some(_blocks) = self.branch("Blocks", true) {
            for name in &self.browser.block_definitions {
                self.block_definition(name, actions);
            }
        }
    }
    fn block_definition(&self, name: &str, actions: &mut Vec<GuiShellAction>) {
        let _id = self.ui.push_id(name);
        let position = self.ui.cursor_screen_pos();
        crate::widget_abi::selectable(self.ui, "##block-definition", false, true);
        crate::widget_abi::literal_text(
            self.ui,
            position,
            self.ui.style_color(StyleColor::Text),
            name,
        );
        let Some(_menu) = self.ui.begin_popup_context_item() else {
            return;
        };
        let action = GuiShellAction::PlaceSketchBlock(name.into());
        if self
            .ui
            .menu_item_config("Place Block")
            .enabled(action.dispatchable(self.state))
            .build()
        {
            actions.push(action);
        }
    }
}
