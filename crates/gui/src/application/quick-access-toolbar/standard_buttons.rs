// SPDX-License-Identifier: GPL-2.0-only
//! Draw built-in quick-access commands using their host actions.
use super::*;

impl QuickAccessToolbar {
    pub(super) fn standard_button(
        &mut self,
        ui: &Ui,
        command: StandardCommand,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let label = command.label();
        match command {
            StandardCommand::New => self.requests.new_document |= ui.button(label),
            StandardCommand::Open => self.requests.open_document |= ui.button(label),
            StandardCommand::Home => {
                if ui.button(label) {
                    actions.push(GuiShellAction::ShowHome);
                }
            }
            StandardCommand::Projects => {
                let disabled = ui.begin_disabled(state.transaction_open);
                self.requests.open_projects |= ui.button(label);
                drop(disabled);
            }
            StandardCommand::Save => crate::chrome::command_button(
                ui,
                label,
                GuiShellAction::Save,
                state,
                [0.; 2],
                actions,
            ),
            StandardCommand::Update => crate::chrome::command_button(
                ui,
                label,
                GuiShellAction::UpdatePart,
                state,
                [0.; 2],
                actions,
            ),
            StandardCommand::Select => {
                selection::button(ui, state, actions, &mut self.open_selection)
            }
            StandardCommand::Undo => {
                crate::chrome::command_button(
                    ui,
                    label,
                    GuiShellAction::Undo,
                    state,
                    [0.; 2],
                    actions,
                );
                self.undo_history(ui, state, actions);
            }
            StandardCommand::Redo => crate::chrome::command_button(
                ui,
                label,
                GuiShellAction::Redo,
                state,
                [0.; 2],
                actions,
            ),
            StandardCommand::MaterialBrowser => {
                let disabled =
                    ui.begin_disabled(state.authoring != crate::GuiAuthoringContext::Part);
                self.requests.open_material_browser |= ui.button(label);
                drop(disabled);
            }
            StandardCommand::Materials => {
                materials::button(ui, state, actions, &mut self.open_materials)
            }
            StandardCommand::AppearanceBrowser => {
                let disabled =
                    ui.begin_disabled(state.authoring != crate::GuiAuthoringContext::Part);
                self.requests.open_appearance_browser |= ui.button(label);
                drop(disabled);
            }
            StandardCommand::Parameters => {
                let disabled = ui.begin_disabled(state.document.is_none());
                self.requests.open_parameters |= ui.button(label);
                drop(disabled);
            }
            StandardCommand::Print
            | StandardCommand::RefineAppearance
            | StandardCommand::AppearanceOverride
            | StandardCommand::AdjustAppearance
            | StandardCommand::DesignDoctor => muted_button(ui, label),
            StandardCommand::ClearAppearance => clear_appearance::button(ui, state, actions),
            StandardCommand::Return => {
                crate::chrome::command_button(
                    ui,
                    label,
                    GuiShellAction::FinishSketch,
                    state,
                    [0.; 2],
                    actions,
                );
                if !GuiShellAction::FinishSketch.enabled(state)
                    && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
                {
                    ui.tooltip_text("Return requires an idle sketch edit.");
                }
            }
            StandardCommand::IProperties => crate::chrome::command_button(
                ui,
                label,
                GuiShellAction::OpenIProperties,
                state,
                [0.; 2],
                actions,
            ),
            StandardCommand::Measure => crate::chrome::command_button(
                ui,
                label,
                GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure),
                state,
                [0.; 2],
                actions,
            ),
        }
    }
}
