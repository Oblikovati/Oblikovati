// SPDX-License-Identifier: GPL-2.0-only
//! The command box: what is typed while a running command has no field for it.
use crate::GuiCommandState;
use imgui::{Ui, WindowFlags};

/// What has been typed since the command last had no field for it.
#[derive(Default)]
pub(crate) struct CommandAliasBox {
    text: String,
}
impl CommandAliasBox {
    /// Borrow the existing command buffer for status feedback; example: `alias.text()`.
    pub(crate) fn text(&self) -> Option<&str> {
        (!self.text.is_empty()).then_some(self.text.as_str())
    }

    /// Show what was typed; a command that ends or regains a field closes the box.
    /// Example: `alias.panel(ui, state, &keys.typed)` once per frame.
    pub(crate) fn panel(&mut self, ui: &Ui, state: &GuiCommandState, typed: &str) {
        if !state.takes_command_alias {
            self.text.clear();
            return;
        }
        self.text.push_str(typed);
        if self.text.is_empty() {
            return;
        }
        let [width, height] = ui.io().display_size;
        ui.window("##command-alias")
            .position([width * 0.5, height - 60.], imgui::Condition::Always)
            .flags(
                WindowFlags::NO_DECORATION
                    | WindowFlags::ALWAYS_AUTO_RESIZE
                    | WindowFlags::NO_INPUTS
                    | WindowFlags::NO_SAVED_SETTINGS
                    | WindowFlags::NO_FOCUS_ON_APPEARING
                    | WindowFlags::NO_NAV,
            )
            .build(|| ui.text(&self.text));
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn only_a_nonempty_command_buffer_has_status_feedback() {
        assert_eq!(CommandAliasBox::default().text(), None);
        let alias = CommandAliasBox { text: "86".into() };
        assert_eq!(alias.text(), Some("86"));
    }
}
