// SPDX-License-Identifier: GPL-2.0-only
//! Draw built-in quick-access commands using their host actions.
use super::*;

impl QuickAccessToolbar {
    pub(super) fn standard_button(
        &mut self,
        ui: &Ui,
        command: StandardCommand,
        icons: &crate::icons::GuiRibbonIcons,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let label = command.label();
        match command {
            StandardCommand::New => {
                self.requests.new_document |=
                    icons.compact_button(ui, label, crate::icons::IconGlyph::NewPart)
            }
            StandardCommand::Open => {
                self.requests.open_document |=
                    icons.compact_button(ui, label, crate::icons::IconGlyph::Open)
            }
            StandardCommand::Home => {
                if icons.compact_button(ui, label, crate::icons::IconGlyph::Home) {
                    actions.push(GuiShellAction::ShowHome);
                }
            }
            StandardCommand::Projects => {
                let disabled = ui.begin_disabled(state.transaction_open);
                self.requests.open_projects |=
                    icons.compact_button(ui, label, crate::icons::IconGlyph::Projects);
                drop(disabled);
            }
            StandardCommand::Save => crate::chrome::command_control(
                ui,
                label,
                GuiShellAction::Save,
                state,
                actions,
                || icons.compact_button(ui, label, crate::icons::IconGlyph::Save),
            ),
            StandardCommand::Update => compact_action(
                ui,
                icons,
                label,
                crate::icons::IconGlyph::Update,
                GuiShellAction::UpdatePart,
                state,
                actions,
            ),
            StandardCommand::Select => {
                selection::button(ui, icons, state, actions, &mut self.open_selection)
            }
            StandardCommand::Undo => {
                compact_action(
                    ui,
                    icons,
                    label,
                    crate::icons::IconGlyph::Undo,
                    GuiShellAction::Undo,
                    state,
                    actions,
                );
                self.undo_history(ui, state, actions);
            }
            StandardCommand::Redo => compact_action(
                ui,
                icons,
                label,
                crate::icons::IconGlyph::Redo,
                GuiShellAction::Redo,
                state,
                actions,
            ),
            StandardCommand::MaterialBrowser => {
                let disabled =
                    ui.begin_disabled(state.authoring != crate::GuiAuthoringContext::Part);
                self.requests.open_material_browser |=
                    icons.compact_button(ui, label, crate::icons::IconGlyph::MaterialBrowser);
                drop(disabled);
            }
            StandardCommand::Materials => {
                materials::button(ui, icons, state, actions, &mut self.open_materials)
            }
            StandardCommand::AppearanceBrowser => {
                let disabled =
                    ui.begin_disabled(state.authoring != crate::GuiAuthoringContext::Part);
                self.requests.open_appearance_browser |=
                    icons.compact_button(ui, label, crate::icons::IconGlyph::AppearanceBrowser);
                drop(disabled);
            }
            StandardCommand::Parameters => {
                let disabled = ui.begin_disabled(state.document.is_none());
                self.requests.open_parameters |=
                    icons.compact_button(ui, label, crate::icons::IconGlyph::Parameters);
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

fn compact_action(
    ui: &Ui,
    icons: &crate::icons::GuiRibbonIcons,
    label: &str,
    glyph: crate::icons::IconGlyph,
    action: GuiShellAction,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    let disabled = ui.begin_disabled(!action.enabled(state));
    let clicked = icons.compact_button(ui, label, glyph);
    drop(disabled);
    if clicked && action.dispatchable(state) {
        actions.push(action);
    }
}
