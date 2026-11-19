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
    engine.create_block.register_dockable(registry)?;
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
    engine.extrude_dialog.register_dockable(registry)?;
    engine.blend_dialog.register_dockables(registry)?;
    engine.hole_dialog.register_dockable(registry)?;
    engine.shell_feature_dialog.register_dockable(registry)?;
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
        self.workspace.sync_before_docking(state);
        self.home.sync_before_docking();
        self.document_tabs
            .menu
            .properties
            .sync_before_docking(state);
        self.workspaces.sync_before_docking(state, actions);
    }
    fn prepare_model_docks(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        self.blend_dialog.sync_before_docking(state, actions);
        self.hole_dialog.sync_before_docking(state, actions);
        self.shell_feature_dialog
            .sync_before_docking(state, actions);
        self.move_body_dialog.sync_before_docking(state, actions);
        self.sketch_pattern.sync_before_docking(state, actions);
        self.metadata.sync_before_docking(state);
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
