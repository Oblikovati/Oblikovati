// SPDX-License-Identifier: GPL-2.0-only
//! Route native dock-tab actions to their window owners.
use crate::application::workspace_layout::panel_tabs::DockPanel;
use crate::widget_abi::DockTabAction;

pub(super) struct DockTabOwners<'a> {
    pub workspace: &'a mut crate::application::workspace_layout::WorkspaceLayout,
    pub material: &'a mut crate::material_browser::MaterialBrowser,
    pub appearance: &'a mut crate::material_browser::MaterialBrowser,
    pub parameters: &'a mut crate::parameters::GuiParameterEditor,
    pub selection: &'a mut crate::selection_filter::SelectionFilterPanel,
    pub history: &'a mut crate::history_browser::HistoryBrowser,
}

pub(super) fn draw(
    ui: &imgui::Ui,
    browser_available: bool,
    workspace_area: Option<crate::application::document_creation::workspaces::WorkspaceArea>,
    owners: DockTabOwners<'_>,
) {
    let tab_action = crate::widget_abi::dock_tab_action(ui);
    let launcher_action = workspace_area
        .filter(|_| browser_available)
        .and_then(|area| owners.workspace.empty_browser_menu(ui, area));
    apply(tab_action.or(launcher_action), browser_available, owners);
}

pub(super) fn apply(
    action: Option<DockTabAction>,
    browser_available: bool,
    owners: DockTabOwners<'_>,
) {
    let Some(action) = action else { return };
    owners.workspace.pending_layout = true;
    match action {
        DockTabAction::ModelBrowser => owners.workspace.panels.open_panel(DockPanel::Model),
        DockTabAction::LogPanel => owners.workspace.panels.open_panel(DockPanel::Log),
        DockTabAction::Favorites => owners.workspace.panels.open_panel(DockPanel::Favorites),
        DockTabAction::MaterialBrowser => owners.material.open(),
        DockTabAction::AppearanceBrowser => owners.appearance.open(),
        DockTabAction::Parameters => owners.parameters.set_open(true),
        DockTabAction::SelectionFilter => owners.selection.open(),
        DockTabAction::HistoryBrowser => owners.history.open(),
        DockTabAction::Search if browser_available => owners.workspace.request_browser_search(),
        DockTabAction::Search => {}
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn opens_the_selected_browser_panel_and_tool() {
        let mut engine = crate::GuiFrameEngine::new().unwrap();
        apply(
            Some(DockTabAction::Favorites),
            true,
            DockTabOwners {
                workspace: &mut engine.workspace,
                material: &mut engine.material_browser,
                appearance: &mut engine.appearance_browser,
                parameters: &mut engine.parameter_editor,
                selection: &mut engine.selection_filter,
                history: &mut engine.history_browser,
            },
        );
        assert!(engine.workspace.panels.favorites_visible());
        assert!(engine.workspace.pending_layout);
        apply(
            Some(DockTabAction::AppearanceBrowser),
            true,
            DockTabOwners {
                workspace: &mut engine.workspace,
                material: &mut engine.material_browser,
                appearance: &mut engine.appearance_browser,
                parameters: &mut engine.parameter_editor,
                selection: &mut engine.selection_filter,
                history: &mut engine.history_browser,
            },
        );
        assert!(engine.appearance_browser.is_open());
    }
}
