// SPDX-License-Identifier: GPL-2.0-only
//! Restore and capture the desktop workspace arrangement.
use super::{GuiFrameEngine, GuiFrameRequest, GuiInputEvent};
use crate::application::workspace_docking::persistence::WorkspaceDockSettings;

impl GuiFrameEngine {
    pub(super) fn sync_tool_visibility(&mut self) {
        self.workspace.tool_visibility = crate::application::workspace_layout::ToolVisibility {
            material: self.material_browser.is_open(),
            appearance: self.appearance_browser.is_open(),
            parameters: self.parameter_editor.is_open(),
            selection_filter: self.selection_filter.is_open(),
            history: self.history_browser.is_open(),
        };
    }

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
        self.workspace
            .set_auto_expand_representations(settings.auto_expand_representations);
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
        if let Some(browser) = &settings.browser_panels {
            self.workspace.panels.restore(browser);
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
        settings.browser_panels = Some(self.workspace.panels.settings());
        settings.auto_expand_representations = self.workspace.auto_expand_representations();
        settings.encode()
    }
}

pub(super) fn workspace_pointer(context: &imgui::Context, request: &GuiFrameRequest) -> [f32; 2] {
    request
        .events
        .iter()
        .rev()
        .find_map(|event| match event {
            GuiInputEvent::PointerMoved(position) => Some(std::array::from_fn(|axis| {
                position[axis] / request.layout_scale()[axis]
            })),
            _ => None,
        })
        .unwrap_or(context.io().mouse_pos)
}

#[cfg(test)]
mod tests {
    #[test]
    fn visibility_snapshot_tracks_each_tool_after_open_and_close() {
        let mut engine = crate::GuiFrameEngine::new().unwrap();
        engine.material_browser.set_open(true);
        engine.appearance_browser.set_open(true);
        engine.parameter_editor.set_open(true);
        engine.selection_filter.set_open(true);
        engine.history_browser.set_open(true);
        engine.sync_tool_visibility();
        let visible = engine.workspace.tool_visibility;
        assert!(visible.material && visible.appearance && visible.parameters);
        assert!(visible.selection_filter && visible.history);
        engine.material_browser.set_open(false);
        engine.appearance_browser.set_open(false);
        engine.parameter_editor.set_open(false);
        engine.selection_filter.set_open(false);
        engine.history_browser.set_open(false);
        engine.sync_tool_visibility();
        assert_eq!(engine.workspace.tool_visibility, Default::default());
    }
}
