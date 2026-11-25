// SPDX-License-Identifier: GPL-2.0-only
//! Use explicit saved workspace profiles for native scenarios with arranged panels.
use super::{
    ApplicationWorld, NativeApplication, NativeHeadWindow, native_owner_at, scripted_window,
};
use std::path::Path;

impl ApplicationWorld {
    /// Load a clean temporary profile, for example before a scenario opens its first document.
    pub(in super::super) fn restart_with_preferences(&mut self, preferences: tempfile::TempDir) {
        self.close();
        self._preferences = preferences;
        self.reopen_application();
    }

    /// Start with a saved layout that has no asset panels, for example before testing docking.
    pub(in super::super) fn with_asset_tools_closed() -> Self {
        let preferences = tempfile::tempdir().unwrap();
        write_layout_without_asset_tools(preferences.path());
        Self::open_profile(preferences)
    }

    /// Load a saved layout before the fixture opens a Part, for example before docking a tool.
    pub(in super::super) fn restore_layout_without_asset_tools(&mut self) {
        self.close();
        write_layout_without_asset_tools(self.preferences_directory());
        self.reopen_application();
    }

    /// Reopen the same native profile, for example after closing a window to save its layout.
    pub fn reopen_application(&mut self) {
        self.close();
        let mut native = NativeHeadWindow::open().unwrap();
        native.ribbon_settings_directory = Some(self.preferences_directory().to_owned());
        let owner = native_owner_at(native.wake.clone(), self.preferences_directory()).unwrap();
        self.app = Some(NativeApplication {
            window: scripted_window(native),
            owner,
        });
        self.pump(Vec::new());
    }
}

fn write_layout_without_asset_tools(directory: &Path) {
    let bytes = oblikovati_gui::GuiFrameEngine::new()
        .unwrap()
        .save_workspace_layout()
        .unwrap();
    crate::native::preferences::NativeWorkspaceLayout::new(directory)
        .save(&bytes)
        .unwrap();
}
