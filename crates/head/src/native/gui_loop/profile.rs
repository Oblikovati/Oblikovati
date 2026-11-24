// SPDX-License-Identifier: GPL-2.0-only
use super::NativeGui;
use crate::native::preferences::{
    NativeRibbonSettings, NativeTemplateSelection, NativeWorkspaceLayout,
};
use std::path::Path;

impl NativeGui {
    pub(in crate::native) fn load_profile(&mut self, directory: &Path) -> Result<(), String> {
        self.engine
            .set_ribbon_settings_store(Box::new(NativeRibbonSettings::new(directory)))?;
        if let Some(bytes) = NativeWorkspaceLayout::new(directory).load()? {
            self.engine.load_workspace_layout(&bytes)?;
        }
        self.engine
            .remember_project_template(NativeTemplateSelection::new(directory).load()?);
        Ok(())
    }

    pub(in crate::native) fn save_profile(&mut self, directory: &Path) -> Result<(), String> {
        let arrangement = self
            .save_workspace_layout()
            .and_then(|bytes| NativeWorkspaceLayout::new(directory).save(&bytes));
        let selection =
            NativeTemplateSelection::new(directory).save(self.engine.project_template_path());
        arrangement.and(selection)
    }
}
