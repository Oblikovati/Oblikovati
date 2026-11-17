// SPDX-License-Identifier: GPL-2.0-only
//! Restore and capture the desktop workspace arrangement.
use super::GuiFrameEngine;
use crate::application::workspace_docking::persistence::WorkspaceDockSettings;

impl GuiFrameEngine {
    /// Restore workspace docking before the first frame.
    /// Example: `engine.load_workspace_layout(&saved_bytes)?` at desktop startup.
    pub fn load_workspace_layout(&mut self, bytes: &[u8]) -> Result<(), String> {
        let settings = WorkspaceDockSettings::decode(bytes)?;
        if !settings.imgui_ini.is_empty() {
            self.context.load_ini_settings(&settings.imgui_ini);
        }
        self.restore_workspace_panels(&settings);
        Ok(())
    }

    fn restore_workspace_panels(&mut self, settings: &WorkspaceDockSettings) {
        if settings.material_browser {
            self.material_browser.open();
        }
        if settings.appearance_browser {
            self.appearance_browser.open();
        }
        self.parameter_editor.set_open(settings.parameters);
        if settings.selection_filter {
            self.selection_filter.open();
        }
        if settings.history_browser {
            self.history_browser.open();
        }
    }

    /// Capture the current workspace arrangement for atomic host storage.
    /// Example: `let bytes = engine.save_workspace_layout()?` before GUI shutdown.
    pub fn save_workspace_layout(&mut self) -> Result<Vec<u8>, String> {
        let mut ini = String::new();
        self.context.save_ini_settings(&mut ini);
        let mut settings = WorkspaceDockSettings::new(ini);
        settings.material_browser = self.material_browser.is_open();
        settings.appearance_browser = self.appearance_browser.is_open();
        settings.parameters = self.parameter_editor.is_open();
        settings.selection_filter = self.selection_filter.is_open();
        settings.history_browser = self.history_browser.is_open();
        settings.encode()
    }
}
