// SPDX-License-Identifier: GPL-2.0-only
//! Versioned, bounded storage for workspace docking and persistent tool visibility.
use serde::{Deserialize, Serialize};

const MAX_LAYOUT_BYTES: usize = 1024 * 1024;
const VERSION: u8 = 1;

#[derive(Debug, Default, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct WorkspaceDockSettings {
    pub version: u8,
    pub imgui_ini: String,
    pub material_browser: bool,
    pub appearance_browser: bool,
    pub parameters: bool,
    pub selection_filter: bool,
    pub history_browser: bool,
    #[serde(default = "enabled_by_default")]
    pub view_cube_visible: bool,
    #[serde(default = "enabled_by_default")]
    pub navigation_bar_visible: bool,
    #[serde(default = "enabled_by_default")]
    pub status_bar_visible: bool,
    #[serde(default = "enabled_by_default")]
    pub auto_expand_representations: bool,
    #[serde(default)]
    pub browser_panels:
        Option<crate::application::workspace_layout::panel_tabs::BrowserPanelSettings>,
}

fn enabled_by_default() -> bool {
    true
}

impl WorkspaceDockSettings {
    pub(crate) fn decode(bytes: &[u8]) -> Result<Self, String> {
        if bytes.len() > MAX_LAYOUT_BYTES {
            return Err(format!(
                "workspace layout {} bytes: expected at most {MAX_LAYOUT_BYTES}",
                bytes.len()
            ));
        }
        let value: Self = serde_json::from_slice(bytes)
            .map_err(|cause| format!("workspace layout: expected versioned JSON: {cause}"))?;
        value.validate()?;
        Ok(value)
    }

    pub(crate) fn encode(&self) -> Result<Vec<u8>, String> {
        self.validate()?;
        let bytes = serde_json::to_vec(self).map_err(|cause| {
            format!("workspace layout: expected serializable settings: {cause}")
        })?;
        if bytes.len() > MAX_LAYOUT_BYTES {
            return Err(format!(
                "workspace layout {} bytes: expected at most {MAX_LAYOUT_BYTES}",
                bytes.len()
            ));
        }
        Ok(bytes)
    }

    fn validate(&self) -> Result<(), String> {
        if self.version != VERSION || self.imgui_ini.contains('\0') {
            return Err(format!(
                "workspace layout version {}, NUL={}: expected version {VERSION} and NUL-free settings",
                self.version,
                self.imgui_ini.contains('\0')
            ));
        }
        Ok(())
    }

    pub(crate) fn new(imgui_ini: String) -> Self {
        Self {
            version: VERSION,
            imgui_ini,
            auto_expand_representations: enabled_by_default(),
            view_cube_visible: true,
            navigation_bar_visible: true,
            status_bar_visible: true,
            ..Self::default()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn settings_round_trip_and_reject_invalid_input() {
        let mut saved = WorkspaceDockSettings::new("[Docking][Data]\n".into());
        saved.material_browser = true;
        let restored = WorkspaceDockSettings::decode(&saved.encode().unwrap()).unwrap();
        assert!(restored.material_browser);
        assert_eq!(restored.imgui_ini, saved.imgui_ini);
        saved.version = 2;
        assert!(saved.encode().is_err());
        saved.version = 1;
        saved.imgui_ini.push('\0');
        assert!(saved.encode().is_err());
        assert!(WorkspaceDockSettings::decode(&vec![b'x'; MAX_LAYOUT_BYTES + 1]).is_err());
        assert!(WorkspaceDockSettings::decode(b"not-json").is_err());
    }

    #[test]
    fn engine_restores_open_panels_and_rejects_invalid_settings_atomically() {
        let mut settings = WorkspaceDockSettings::new(String::new());
        settings.material_browser = true;
        settings.selection_filter = true;
        let mut engine = crate::GuiFrameEngine::new().unwrap();
        engine
            .load_workspace_layout(&settings.encode().unwrap())
            .unwrap();
        assert!(engine.load_workspace_layout(b"invalid").is_err());
        let restored =
            WorkspaceDockSettings::decode(&engine.save_workspace_layout().unwrap()).unwrap();
        assert!(restored.material_browser && restored.selection_filter);
        assert!(!restored.appearance_browser && !restored.parameters && !restored.history_browser);
    }
    #[test]
    fn view_controls_default_visible_and_round_trip_independently() {
        let legacy = br#"{"version":1,"imgui_ini":"","material_browser":false,"appearance_browser":false,"parameters":false,"selection_filter":false,"history_browser":false}"#;
        let mut settings = WorkspaceDockSettings::decode(legacy).unwrap();
        assert!(
            settings.view_cube_visible
                && settings.navigation_bar_visible
                && settings.status_bar_visible
        );
        settings.status_bar_visible = false;
        settings.view_cube_visible = false;
        let mut engine = crate::GuiFrameEngine::new().unwrap();
        engine
            .load_workspace_layout(&settings.encode().unwrap())
            .unwrap();
        let saved =
            WorkspaceDockSettings::decode(&engine.save_workspace_layout().unwrap()).unwrap();
        assert!(
            !saved.view_cube_visible && saved.navigation_bar_visible && !saved.status_bar_visible
        );
        settings.status_bar_visible = true;
        settings.view_cube_visible = true;
        settings.navigation_bar_visible = false;
        engine
            .load_workspace_layout(&settings.encode().unwrap())
            .unwrap();
        let saved =
            WorkspaceDockSettings::decode(&engine.save_workspace_layout().unwrap()).unwrap();
        assert!(
            saved.view_cube_visible && !saved.navigation_bar_visible && saved.status_bar_visible
        );
    }
    #[test]
    fn old_settings_default_and_round_trip_representation_preference() {
        let legacy = br#"{"version":1,"imgui_ini":"","material_browser":false,"appearance_browser":false,"parameters":false,"selection_filter":false,"history_browser":false}"#;
        let mut settings = WorkspaceDockSettings::decode(legacy).unwrap();
        assert!(settings.auto_expand_representations);
        settings.auto_expand_representations = false;
        let mut engine = crate::GuiFrameEngine::new().unwrap();
        engine
            .load_workspace_layout(&settings.encode().unwrap())
            .unwrap();
        let restored =
            WorkspaceDockSettings::decode(&engine.save_workspace_layout().unwrap()).unwrap();
        assert!(!restored.auto_expand_representations);
    }
}
