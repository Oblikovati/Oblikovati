// SPDX-License-Identifier: GPL-2.0-only
//! Command dispatch and history labels.
use super::*;
impl GuiShellAction {
    /// Keep available history controls inert during an edit; example: `action.dispatchable(&state)` after a click.
    pub fn dispatchable(&self, state: &GuiCommandState) -> bool {
        let history = matches!(self, Self::Undo | Self::Redo | Self::EditSketch { .. });
        self.enabled(state) && !(history && state.transaction_open)
    }
    pub(crate) fn next_label<'a>(&self, state: &'a GuiCommandState) -> Option<&'a str> {
        match self {
            Self::Undo => state.undo_label.as_deref(),
            Self::Redo => state.redo_label.as_deref(),
            _ => None,
        }
    }
}
