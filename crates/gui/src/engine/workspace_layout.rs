// SPDX-License-Identifier: GPL-2.0-only
//! Restore and capture the desktop workspace arrangement.
use super::{GuiFrameEngine, GuiFrameRequest, GuiInputEvent};
use crate::application::workspace_docking::persistence::WorkspaceDockSettings;

impl GuiFrameEngine {
    /// Keep a window's picture for its tab; example: after each completed presentation.
    pub fn note_presented_frame(&mut self, extent: [u32; 2], rgba: &[u8]) {
        self.document_tabs.previews.note_frame(extent, rgba);
    }
    /// Report the visible workspace for the native window title.
    pub fn home_in_front(&self) -> bool {
        self.home.active
    }
    /// Update the workspace and report whether it needs the fresh profile's asset dock.
    /// Example: `let seed_asset_dock = engine.prepare_workspace_frame(&request.commands)`.
    pub(super) fn prepare_workspace_frame(&mut self, commands: &crate::GuiCommandState) -> bool {
        let first_home = self.home.observe_document(commands.document);
        self.workspace.pending_layout = self.workspace.document_dock_tabs.synchronize(
            &self.context,
            commands,
            self.home.active,
        );
        self.workspace.part_context_open = self.part_context_open && commands.model_face_chosen;
        let seed_asset_dock = first_home && !self.workspace_layout_restored;
        if seed_asset_dock {
            self.material_browser.open();
            self.appearance_browser.open();
        }
        seed_asset_dock
    }

    pub(super) fn sync_tool_visibility(&mut self) {
        self.workspace.tool_visibility = crate::application::workspace_layout::ToolVisibility {
            material: self.material_browser.is_requested(),
            appearance: self.appearance_browser.is_requested(),
            parameters: self.parameter_editor.is_open(),
            selection_filter: self.selection_filter.is_open(),
            history: self.history_browser.is_open(),
        };
    }

    pub(super) fn workspace_needs_frame(&self, home_was_active: bool) -> bool {
        self.extrude_dialog.needs_frame()
            || self.orientation.needs_frame()
            || self.ribbon.panels.pending_layout
            || self.ribbon.needs_frame()
            || self.ribbon.customization.needs_frame()
            || self.workspace.pending_layout
            || self.home.active != home_was_active
            || self.home.hover_changed
            || self.document_files.needs_frame()
            || self.document_tabs.needs_frame()
            || self.workspaces.needs_frame()
            || self.workspace.switcher.needs_frame()
            || self.document_tabs.menu.properties.needs_frame()
    }

    /// Restore workspace docking before the first frame.
    /// Example: `engine.load_workspace_layout(&saved_bytes)?` at desktop startup.
    pub fn load_workspace_layout(&mut self, bytes: &[u8]) -> Result<(), String> {
        let settings = WorkspaceDockSettings::decode(bytes)?;
        if !settings.imgui_ini.is_empty() {
            self.context.load_ini_settings(&settings.imgui_ini);
        }
        self.restore_workspace_panels(&settings);
        self.workspace_layout_restored = true;
        Ok(())
    }

    fn restore_workspace_panels(&mut self, settings: &WorkspaceDockSettings) {
        self.workspace.view_cube_visible = settings.view_cube_visible;
        self.workspace.navigation_bar_visible = settings.navigation_bar_visible;
        self.workspace.status_bar_visible = settings.status_bar_visible;
        self.workspace.marking_menu_enabled = settings.marking_menu_enabled;
        self.workspace.selection_toolbar_visible = settings.selection_toolbar_visible;
        self.workspace.outer_dock_anchor = settings.outer_dock_anchor.unwrap_or_default();
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
        settings.material_browser = self.material_browser.is_requested();
        settings.appearance_browser = self.appearance_browser.is_requested();
        settings.parameters = self.parameter_editor.is_open();
        settings.selection_filter = self.selection_filter.is_open();
        settings.history_browser = self.history_browser.is_open();
        settings.browser_panels = Some(self.workspace.panels.settings());
        settings.outer_dock_anchor =
            Some(self.workspace.outer_dock_anchor).filter(|anchor| *anchor != 0);
        settings.auto_expand_representations = self.workspace.auto_expand_representations();
        settings.view_cube_visible = self.workspace.view_cube_visible;
        settings.navigation_bar_visible = self.workspace.navigation_bar_visible;
        settings.status_bar_visible = self.workspace.status_bar_visible;
        settings.marking_menu_enabled = self.workspace.marking_menu_enabled;
        settings.selection_toolbar_visible = self.workspace.selection_toolbar_visible;
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

/// Home is left by actions that put a document in front of it.
pub(super) fn home_follows_actions(
    home: &mut crate::application::home_recent_documents::HomeWorkspace,
    selected: &[crate::GuiShellAction],
) {
    for action in selected {
        match action {
            crate::GuiShellAction::ShowHome => home.active = true,
            crate::GuiShellAction::CloseDocumentWindow(_) if home.reached_from_home => {
                home.active = true;
            }
            crate::GuiShellAction::ActivateDocument(_)
            | crate::GuiShellAction::ActivateWindow(_)
            | crate::GuiShellAction::OpenDocument(_)
            | crate::GuiShellAction::OpenDeferredDrawing(_)
            | crate::GuiShellAction::CreateDocument(_)
            | crate::GuiShellAction::CreateProjectTemplate(_)
            | crate::GuiShellAction::DrawingLayout(
                crate::GuiDrawingLayoutAction::Confirm { .. }
                | crate::GuiDrawingLayoutAction::Acknowledge { .. },
            )
            | crate::GuiShellAction::NewPart => home.active = false,
            _ => {}
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn restored_settings_keep_assets_closed_on_the_first_home_frame() {
        let mut engine = GuiFrameEngine::new().unwrap();
        let settings = WorkspaceDockSettings::new(String::new()).encode().unwrap();
        engine.load_workspace_layout(&settings).unwrap();
        assert!(!engine.prepare_workspace_frame(&crate::GuiCommandState::default()));
        assert!(engine.home.active);
        assert!(!engine.material_browser.is_open());
        assert!(!engine.appearance_browser.is_open());
    }

    #[test]
    fn side_docks_keep_the_viewport_wide_and_recover_their_preferred_widths() {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = GuiFrameRequest {
            extent: [1440, 900],
            events: Vec::new(),
            commands: Default::default(),
            status: None,
            delta_seconds: 1. / 60.,
            framebuffer_scale: [1.; 2],
        };
        engine.frame(&request).unwrap();
        request.commands.document = Some(1);
        request.commands.authoring = crate::GuiAuthoringContext::Part;
        for width in [1440, 1100, 780, 1440] {
            request.extent[0] = width;
            let mut frame = engine.frame(&request).unwrap();
            for _ in 0..3 {
                frame = engine.frame(&request).unwrap();
            }
            let [left, _, right, _] = frame.document_area.unwrap();
            assert!(
                right - left >= (width as f32 * 0.55) as u32,
                "{width}: {left}..{right}"
            );
            if width == 1440 {
                assert!(left.abs_diff(width / 4) <= 3, "Model dock edge: {left}");
                assert!((259..=263).contains(&(width - right)), "{right}");
            }
        }
    }

    #[test]
    fn home_in_front_of_a_part_releases_and_restores_its_dock_space() {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = GuiFrameRequest {
            extent: [1100, 760],
            events: Vec::new(),
            commands: Default::default(),
            status: None,
            delta_seconds: 1. / 60.,
            framebuffer_scale: [1.; 2],
        };
        engine.frame(&request).unwrap();
        request.commands.document = Some(1);
        request.commands.authoring = crate::GuiAuthoringContext::Part;
        let mut bounds = Vec::new();
        for home in [false, true, false] {
            engine.home.active = home;
            let area = (0..4).map(|_| engine.frame(&request).unwrap().document_area);
            bounds.push(area.last().unwrap().unwrap());
            assert_eq!(engine.workspace.panels.model_visible(), !home);
            assert_eq!(engine.appearance_browser.is_open(), !home);
        }
        assert_eq!(bounds[0], bounds[2]);
        assert_eq!([bounds[1][0], bounds[1][2]], [0, 1100]);
        assert_eq!(bounds[1][1], bounds[0][1]);
    }

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
