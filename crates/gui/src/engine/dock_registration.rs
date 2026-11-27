// SPDX-License-Identifier: GPL-2.0-only
//! Register each owner with the GUI-owned dock lifecycle.
use super::{GuiCommandState, GuiFrameEngine, GuiShellAction};

pub(super) fn register(engine: &mut GuiFrameEngine) -> Result<(), String> {
    register_application_windows(engine)?;
    register_model_windows(engine)
}
fn register_application_windows(engine: &mut GuiFrameEngine) -> Result<(), String> {
    let registry = &mut engine.dock_lifecycle;
    engine.material_browser.register_dockable(registry)?;
    engine.appearance_browser.register_dockable(registry)?;
    engine.parameter_editor.register_dockables(registry)?;
    engine.ribbon.customization.register_dockable(registry)?;
    engine.ribbon.team_web.register_dockable(registry)?;
    engine.create_block.register_dockable(registry)?;
    engine.sketch_text.register_dockable(registry)?;
    engine.curvature_settings.register_dockable(registry)?;
    engine.inference_scope.register_dockable(registry)?;
    engine.preferences.register_dockable(registry)?;
    engine.precise_input.register_dockable(registry)?;
    engine.orientation.register_dockables(registry)?;
    engine.workspace.register_dockables(registry)?;
    engine.external_docks.register_dockables(registry)?;
    engine.workspaces.register_dockables(registry)?;
    Ok(())
}
fn register_model_windows(engine: &mut GuiFrameEngine) -> Result<(), String> {
    let registry = &mut engine.dock_lifecycle;
    engine.document_files.register_dockables(registry)?;
    engine.home.register_dockables(registry)?;
    engine
        .document_tabs
        .menu
        .properties
        .register_dockable(registry)?;
    engine.dimension_value_editor.register_dockables(registry)?;
    engine.sketch_pattern.register_dockable(registry)?;
    engine.theme_editor.register_dockable(registry)?;
    engine.clear_appearance.register_dockable(registry)?;
    engine.offset_plane.register_dockable(registry)?;
    engine.metadata.register_dockable(registry)?;
    engine.model_state_table.register_dockable(registry)?;
    engine.extrude_dialog.register_dockable(registry)?;
    engine.blend_dialog.register_dockables(registry)?;
    engine.hole_dialog.register_dockable(registry)?;
    engine.shell_feature_dialog.register_dockable(registry)?;
    engine.sweep_dialog.register_dockable(registry)?;
    engine.revolve_dialog.register_dockable(registry)?;
    engine.loft_dialog.register_dockable(registry)?;
    engine.move_body_dialog.register_dockable(registry)?;
    engine.history_browser.register_dockable(registry)?;
    engine.selection_filter.register_dockable(registry)
}

#[cfg(test)]
pub(crate) fn owner_types(engine: &GuiFrameEngine) -> std::collections::BTreeSet<&'static str> {
    engine.dock_lifecycle.owner_types()
}

#[cfg(test)]
pub(crate) fn owner_instance_counts(
    engine: &GuiFrameEngine,
) -> std::collections::BTreeMap<&'static str, usize> {
    engine.dock_lifecycle.owner_instance_counts()
}

impl GuiFrameEngine {
    pub(super) fn draw_asset_docks(
        (material, appearance): (
            &mut crate::material_browser::MaterialBrowser,
            &mut crate::material_browser::MaterialBrowser,
        ),
        ui: &imgui::Ui,
        state: &GuiCommandState,
        home_active: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        material.panel(ui, state, home_active, actions);
        appearance.panel(ui, state, home_active, actions);
    }

    pub(super) fn dock_editor_consumed_escape(&self) -> bool {
        self.sketch_text.escape_consumed() || self.dimension_value_editor.had_keyboard()
    }

    pub(super) fn prepare_dock_owners(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.prepare_application_docks(state, actions);
        self.prepare_model_docks(state, actions);
    }
    fn prepare_application_docks(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.ribbon.customization.sync_before_docking(state);
        self.create_block.sync_before_docking(state, actions);
        self.sketch_text.synchronize(state);
        self.workspace.sync_before_docking(state, self.home.active);
        self.home.sync_before_docking();
        self.material_browser
            .sync_before_docking(state, self.home.active);
        self.appearance_browser
            .sync_before_docking(state, self.home.active);
        self.document_tabs
            .menu
            .properties
            .sync_before_docking(state);
        self.workspaces.sync_before_docking(state, actions);
    }
    fn prepare_model_docks(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        self.revolve_dialog.synchronize(state, actions);
        self.sweep_dialog.synchronize(state, actions);
        self.loft_dialog.synchronize(state, actions);
        self.blend_dialog.sync_before_docking(state, actions);
        self.hole_dialog.sync_before_docking(state, actions);
        self.shell_feature_dialog
            .sync_before_docking(state, actions);
        self.move_body_dialog.sync_before_docking(state, actions);
        self.sketch_pattern.sync_before_docking(state, actions);
        self.metadata.sync_before_docking(state);
        self.model_state_table.sync_before_docking(state);
        self.clear_appearance.sync_before_docking(state);
        self.offset_plane.sync_before_docking(state);
        self.extrude_dialog.sync_before_docking(state);
        self.dimension_value_editor.sync_before_docking(state);
        self.precise_input.sync_before_docking(state);
        self.orientation.set_suspended(self.home.active);
        self.preferences
            .sync_before_docking(state, self.home.active);
    }
}
impl GuiFrameEngine {
    pub(super) fn draw_sketch_editors(
        (text, curvature): (
            &mut crate::sketch::text::SketchTextDialog,
            &mut crate::sketch::curvature::CurvatureSettingsWindow,
        ),
        ui: &imgui::Ui,
        commands: &crate::GuiCommandState,
        actions: &mut Vec<crate::GuiShellAction>,
    ) {
        text.panel(ui, commands, actions);
        curvature.panel(ui, commands.curvature_settings, actions);
    }
}
