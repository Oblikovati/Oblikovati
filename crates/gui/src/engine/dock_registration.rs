// SPDX-License-Identifier: GPL-2.0-only
//! Register each owner with the GUI-owned dock lifecycle.
use super::GuiFrameEngine;

pub(super) fn register(engine: &mut GuiFrameEngine) -> Result<(), String> {
    let registry = &mut engine.dock_lifecycle;
    engine.material_browser.register_dockable(registry)?;
    engine.appearance_browser.register_dockable(registry)?;
    engine.parameter_editor.register_dockables(registry)?;
    engine.ribbon.customization.register_dockable(registry)?;
    engine.create_block.register_dockable(registry)?;
    engine.preferences.register_dockable(registry)?;
    engine.precise_input.register_dockable(registry)?;
    engine.orientation.register_dockables(registry)?;
    engine.workspace.register_dockables(registry)?;
    engine.external_docks.register_dockables(registry)?;
    engine.workspaces.register_dockables(registry)?;
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
