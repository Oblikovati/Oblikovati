// SPDX-License-Identifier: GPL-2.0-only
use super::NativeGui;
#[cfg(any(target_os = "linux", target_os = "windows"))]
use crate::native::preferences::{
    NativeRibbonSettings, NativeSweepPresets, NativeTemplateSelection, NativeWorkspaceLayout,
};
use std::path::Path;

impl NativeGui {
    #[cfg(any(target_os = "linux", target_os = "windows"))]
    pub(in crate::native) fn load_profile(&mut self, directory: &Path) -> Result<(), String> {
        self.engine
            .set_ribbon_settings_store(Box::new(NativeRibbonSettings::new(directory)))?;
        self.engine
            .set_sweep_preset_store(Box::new(NativeSweepPresets::new(directory)))?;
        if let Some(bytes) = NativeWorkspaceLayout::new(directory).load()? {
            self.engine.load_workspace_layout(&bytes)?;
        }
        self.engine
            .remember_project_template(NativeTemplateSelection::new(directory).load()?);
        Ok(())
    }

    #[cfg(target_arch = "wasm32")]
    pub(in crate::native) fn load_profile(&mut self, directory: &Path) -> Result<(), String> {
        let _ = directory;
        Ok(())
    }

    #[cfg(any(target_os = "linux", target_os = "windows"))]
    pub(in crate::native) fn save_profile(&mut self, directory: &Path) -> Result<(), String> {
        let arrangement = self
            .save_workspace_layout()
            .and_then(|bytes| NativeWorkspaceLayout::new(directory).save(&bytes));
        let selection =
            NativeTemplateSelection::new(directory).save(self.engine.project_template_path());
        arrangement.and(selection)
    }

    #[cfg(target_arch = "wasm32")]
    pub(in crate::native) fn save_profile(&mut self, directory: &Path) -> Result<(), String> {
        let _ = directory;
        Ok(())
    }
}
