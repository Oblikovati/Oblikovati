// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiShellAction, GuiViewCommand};

impl GuiShellAction {
    /// Resolve a desktop command ID; example: `GuiShellAction::from_command_id("View.Home")`.
    pub fn from_command_id(id: &str) -> Option<Self> {
        match id {
            "View.Home" => Some(Self::Home),
            "View.ZoomAll" => Some(Self::ViewCommand(GuiViewCommand::FitAll)),
            "View.LookAt" => Some(Self::LookAtSelection),
            _ => crate::sketch::marking_menu::action(id).map(|(_, action)| action),
        }
    }
}
