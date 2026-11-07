// SPDX-License-Identifier: GPL-2.0-only
//! The model-state branch menu.
use super::*;

impl BrowserTree<'_> {
    pub(super) fn model_state_menu(
        &self,
        states: &oblikovati_model::model_state::ModelStateMembers,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(_menu) = self.ui.begin_popup("##model-state-menu") else {
            return;
        };
        self.ui.menu("New Model State", || {
            if self.ui.menu_item("New Model State") {
                actions.push(GuiShellAction::BrowserChoice {
                    document: self.state.document.unwrap_or(0),
                    choice: crate::GuiBrowserChoice::CreateModelState,
                });
            }
            if let Some(active) = states.active() {
                let label = format!("Duplicate {}", active.name);
                let cursor = self.ui.cursor_pos();
                let width = self.ui.calc_text_size(&label)[0];
                crate::widget_abi::dummy(self.ui, [width, 0.]);
                crate::widget_abi::cursor_position(self.ui, cursor);
                let position = self.ui.cursor_screen_pos();
                let chosen = crate::widget_abi::selectable(
                    self.ui,
                    "##duplicate-active-model-state",
                    false,
                    false,
                );
                crate::widget_abi::literal_text(
                    self.ui,
                    position,
                    self.ui.style_color(StyleColor::Text),
                    &label,
                );
                if chosen {
                    actions.push(GuiShellAction::BrowserChoice {
                        document: self.state.document.unwrap_or(0),
                        choice: crate::GuiBrowserChoice::CopyModelState(active.id),
                    });
                }
            }
        });
    }
}
