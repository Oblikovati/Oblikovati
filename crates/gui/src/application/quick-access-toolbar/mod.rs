// SPDX-License-Identifier: GPL-2.0-only
//! The customizable command row: which commands each document kind shows, and where the row sits.
mod clear_appearance;
mod materials;
mod standard_buttons;
mod standard_commands;
use standard_commands::{STANDARD, StandardCommand};
#[path = "plane_selection.rs"]
pub(crate) mod plane_selection;
mod selection;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, Ui};
use oblikovati_api::types::DocumentType;
use std::collections::BTreeMap;

const CUSTOMIZE_MENU: &str = "##quick-access-customize";
const SHORTCUT_MENU: &str = "##quick-access-shortcut";
const UNDO_HISTORY: &str = "##undo-history";
const NOT_AVAILABLE: &str = "This command is not available in this build.";
/// Height of the separate row below the ribbon.
pub(crate) const BELOW_ROW_HEIGHT: f32 = 30.;
const SEPARATOR_WIDTH: f32 = 7.;

/// One thing a person added from a ribbon command's menu.
#[derive(Clone, Debug, PartialEq, Eq)]
enum AddedEntry {
    Command(String),
    Separator,
}
/// What one document kind shows; every document of the kind shares it.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
struct KindSelection {
    /// Standard commands whose choice differs from the default.
    toggled: Vec<StandardCommand>,
    added: Vec<AddedEntry>,
}
impl KindSelection {
    fn shows(&self, command: StandardCommand) -> bool {
        command.default_shown() != self.toggled.contains(&command)
    }
    fn toggle(&mut self, command: StandardCommand) {
        match self.toggled.iter().position(|known| *known == command) {
            Some(index) => {
                self.toggled.remove(index);
            }
            None => self.toggled.push(command),
        }
    }
    fn has_command(&self, label: &str) -> bool {
        self.added
            .iter()
            .any(|entry| matches!(entry, AddedEntry::Command(known) if known == label))
    }
    fn remove_command(&mut self, label: &str) {
        self.added
            .retain(|entry| !matches!(entry, AddedEntry::Command(known) if known == label));
    }
}
/// Requests only the frame's owner can carry out.
#[derive(Default)]
pub(crate) struct QuickAccessRequests {
    pub new_document: bool,
    pub open_document: bool,
    pub choose_plane: bool,
    pub choose_plane_for_extrude: bool,
    pub open_extrude: bool,
    pub open_material_browser: bool,
    pub open_appearance_browser: bool,
    pub open_projects: bool,
    pub open_parameters: bool,
}
/// The toolbar's choices for this session; a later scenario may ask for them to persist.
#[derive(Default)]
pub(crate) struct QuickAccessToolbar {
    #[cfg(test)]
    pub(crate) test_save_bounds: Option<[[f32; 2]; 2]>,
    kinds: BTreeMap<String, KindSelection>,
    pub below_ribbon: bool,
    shortcut_under_menu: Option<String>,
    open_selection: bool,
    open_materials: bool,
    requests: QuickAccessRequests,
    /// A right press landed on a command with its own menu, not on the ribbon behind it.
    pub command_took_right_click: bool,
}
fn kind_key(document_type: Option<DocumentType>) -> String {
    format!("{document_type:?}")
}
impl QuickAccessToolbar {
    pub(crate) fn take_requests(&mut self) -> QuickAccessRequests {
        std::mem::take(&mut self.requests)
    }
    fn selection(&mut self, state: &GuiCommandState) -> &mut KindSelection {
        self.kinds.entry(kind_key(state.document_type)).or_default()
    }
    /// Draw the row, then its control; example: `toolbar.row(ui, state, actions)`.
    pub(crate) fn row(
        &mut self,
        ui: &Ui,
        icons: &crate::icons::GuiRibbonIcons,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        tips: &crate::ribbon_key_tips::RibbonKeyTips,
    ) {
        let selection = self.selection(state).clone();
        for (index, command) in STANDARD
            .into_iter()
            .filter(|command| selection.shows(*command))
            .enumerate()
        {
            self.standard_button(ui, command, icons, state, actions);
            let tip = (index + 1).to_string();
            if tips.tabs() && index < 9 && tips.matching(&tip) {
                crate::ribbon_key_tips::badge(ui, &tip);
            }
            ui.same_line();
        }
        let mut added_index = STANDARD
            .into_iter()
            .filter(|command| selection.shows(*command))
            .count();
        for entry in &selection.added {
            self.added_entry(ui, entry, state, actions);
            if matches!(entry, AddedEntry::Command(_)) {
                added_index += 1;
                let tip = added_index.to_string();
                if tips.tabs() && added_index <= 9 && tips.matching(&tip) {
                    crate::ribbon_key_tips::badge(ui, &tip);
                }
            }
            ui.same_line();
        }
        self.shortcut_menu(ui, state);
        self.customize_control(ui, state);
    }
    pub(crate) fn activate_tip(
        &mut self,
        index: usize,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let selection = self.selection(state).clone();
        let standard: Vec<_> = STANDARD
            .into_iter()
            .filter(|command| selection.shows(*command))
            .collect();
        if index >= standard.len() {
            self.activate_added_tip(&selection, index - standard.len(), state, actions);
            return;
        }
        let command = standard.get(index).copied();
        let action = match command {
            Some(StandardCommand::New) => {
                self.requests.new_document = true;
                return;
            }
            Some(StandardCommand::Open) => {
                self.requests.open_document = true;
                return;
            }
            Some(StandardCommand::Save) => GuiShellAction::Save,
            Some(StandardCommand::Undo) => GuiShellAction::Undo,
            Some(StandardCommand::Redo) => GuiShellAction::Redo,
            Some(StandardCommand::Home) => GuiShellAction::ShowHome,
            Some(StandardCommand::Update) => GuiShellAction::UpdatePart,
            Some(StandardCommand::IProperties) => GuiShellAction::OpenIProperties,
            Some(StandardCommand::Return) => GuiShellAction::FinishSketch,
            Some(StandardCommand::ClearAppearance) => {
                let Some(action) = clear_appearance::action(state) else {
                    return;
                };
                action
            }
            Some(StandardCommand::Measure) => {
                GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure)
            }
            Some(StandardCommand::Select)
                if GuiShellAction::SetSelectionMode(crate::GuiSelectionMode::All)
                    .enabled(state) =>
            {
                self.open_selection = true;
                return;
            }
            Some(StandardCommand::Projects) if !state.transaction_open => {
                self.requests.open_projects = true;
                return;
            }
            Some(StandardCommand::MaterialBrowser)
                if state.authoring == crate::GuiAuthoringContext::Part =>
            {
                self.requests.open_material_browser = true;
                return;
            }
            Some(StandardCommand::Materials)
                if state.authoring == crate::GuiAuthoringContext::Part
                    && !state.transaction_open =>
            {
                self.open_materials = true;
                return;
            }
            Some(StandardCommand::AppearanceBrowser)
                if state.authoring == crate::GuiAuthoringContext::Part =>
            {
                self.requests.open_appearance_browser = true;
                return;
            }
            Some(StandardCommand::Parameters) if state.document.is_some() => {
                self.requests.open_parameters = true;
                return;
            }
            _ => return,
        };
        if action.enabled(state) {
            actions.push(action);
        }
    }
    fn activate_added_tip(
        &self,
        selection: &KindSelection,
        index: usize,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let label = selection
            .added
            .iter()
            .filter_map(|entry| match entry {
                AddedEntry::Command(label) => Some(label.as_str()),
                AddedEntry::Separator => None,
            })
            .nth(index);
        if label == Some("Move Bodies") && GuiShellAction::OpenMoveBody.dispatchable(state) {
            actions.push(GuiShellAction::OpenMoveBody);
        }
    }
    /// Right-click on Undo reads what it would undo, latest first.
    fn undo_history(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let hovered = ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED);
        if hovered && ui.is_mouse_clicked(MouseButton::Right) {
            self.command_took_right_click = true;
        }
        if hovered && ui.is_mouse_released(MouseButton::Right) {
            ui.open_popup(UNDO_HISTORY);
        }
        let Some(_menu) = ui.begin_popup(UNDO_HISTORY) else {
            return;
        };
        if ui.menu_item("History Browser…") && GuiShellAction::OpenHistoryBrowser.enabled(state) {
            actions.push(GuiShellAction::OpenHistoryBrowser);
        }
        for label in state.undo_labels.iter().rev() {
            ui.text(label);
        }
        // The last row's descenders must fit inside the popup clip (#4068).
        ui.dummy([0., 2.]);
    }
    fn added_entry(
        &mut self,
        ui: &Ui,
        entry: &AddedEntry,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let AddedEntry::Command(label) = entry else {
            separator(ui);
            return;
        };
        if label == "Move Bodies" {
            let action = GuiShellAction::OpenMoveBody;
            let disabled = ui.begin_disabled(!action.dispatchable(state));
            if ui.button(label) {
                actions.push(action);
            }
            drop(disabled);
        } else if label == EXTRUDE && extrude_can_start(state) {
            if ui.button(label) {
                match crate::extrude_dialog::start_action(state) {
                    GuiShellAction::ChooseSketchPlaneForExtrude => {
                        self.requests.choose_plane_for_extrude = true
                    }
                    GuiShellAction::OpenExtrude => self.requests.open_extrude = true,
                    _ => unreachable!(),
                }
            }
        } else {
            muted_button(ui, label);
        }
        let hovered = ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED);
        if hovered && ui.is_mouse_clicked(MouseButton::Right) {
            self.command_took_right_click = true;
        }
        if hovered && ui.is_mouse_released(MouseButton::Right) {
            self.shortcut_under_menu = Some(label.clone());
            ui.open_popup(SHORTCUT_MENU);
        }
    }
    fn shortcut_menu(&mut self, ui: &Ui, state: &GuiCommandState) {
        let Some(_menu) = ui.begin_popup(SHORTCUT_MENU) else {
            return;
        };
        let Some(label) = self.shortcut_under_menu.clone() else {
            return;
        };
        if ui.menu_item("Remove from Quick Access Toolbar") {
            self.selection(state).remove_command(&label);
        }
        if ui.menu_item("Add Separator") {
            add_separator_after(self.selection(state), &label);
        }
        // Leave room for the last menu row's descenders.
        ui.dummy([0., 2.]);
    }
    fn customize_control(&mut self, ui: &Ui, state: &GuiCommandState) {
        if ui.button("...") {
            ui.open_popup(CUSTOMIZE_MENU);
        }
        let Some(_menu) = ui.begin_popup(CUSTOMIZE_MENU) else {
            return;
        };
        for command in STANDARD {
            let shown = self.selection(state).shows(command);
            if ui
                .menu_item_config(command.label())
                .selected(shown)
                .enabled(command.implemented())
                .build()
            {
                self.selection(state).toggle(command);
            }
            if !command.implemented()
                && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
            {
                ui.tooltip_text(NOT_AVAILABLE);
            }
        }
        ui.separator();
        let placement = if self.below_ribbon {
            "Show Above the Ribbon"
        } else {
            "Show Below the Ribbon"
        };
        if ui.menu_item(placement) {
            self.below_ribbon = !self.below_ribbon;
        }
    }
    pub(crate) fn add_ribbon_command(&mut self, state: &GuiCommandState, label: &str) {
        let selection = self.selection(state);
        if !selection.has_command(label) {
            selection.added.push(AddedEntry::Command(label.to_owned()));
        }
    }
    pub(crate) fn has_ribbon_command(&mut self, state: &GuiCommandState, label: &str) -> bool {
        self.selection(state).has_command(label)
    }
    /// A ribbon command's own menu; example: right-click Extrude on the 3D Model tab.
    pub(crate) fn ribbon_command_menu(
        &mut self,
        ui: &Ui,
        label: &str,
        supported: bool,
        state: &GuiCommandState,
    ) {
        let identity = format!("##ribbon-command-{label}");
        let hovered = ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED);
        if hovered && ui.is_mouse_clicked(MouseButton::Right) {
            self.command_took_right_click = true;
        }
        if hovered && ui.is_mouse_released(MouseButton::Right) {
            ui.open_popup(&identity);
        }
        let Some(_menu) = ui.begin_popup(&identity) else {
            return;
        };
        let present = self.selection(state).has_command(label);
        if ui
            .menu_item_config("Add to Quick Access Toolbar")
            .enabled(supported && !present)
            .build()
        {
            self.selection(state)
                .added
                .push(AddedEntry::Command(label.to_owned()));
        }
    }
    /// Return the height needed by the lower toolbar at the chosen icon scale.
    /// Example: `toolbar.below_height(icons)` before placing the workspace.
    pub(crate) fn below_height(&self, icons: &crate::icons::GuiRibbonIcons) -> f32 {
        BELOW_ROW_HEIGHT.max(icons.cell([24., 24.])[1] + 6.)
    }
    /// Draw the row that takes its own band under the ribbon; example: `toolbar.band(ui, ...)`.
    pub(crate) fn band_below(
        &mut self,
        ui: &Ui,
        bounds: ([f32; 2], f32),
        icons: &crate::icons::GuiRibbonIcons,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        tips: &crate::ribbon_key_tips::RibbonKeyTips,
    ) {
        crate::widget_abi::window_geometry(
            ui,
            bounds.0,
            [bounds.1, self.below_height(icons)],
            imgui::Condition::Always,
        );
        ui.window("##quick-access-below")
            .flags(
                imgui::WindowFlags::NO_DECORATION
                    | imgui::WindowFlags::NO_MOVE
                    | imgui::WindowFlags::NO_DOCKING
                    | imgui::WindowFlags::NO_SAVED_SETTINGS
                    | imgui::WindowFlags::NO_FOCUS_ON_APPEARING,
            )
            .build(|| self.row(ui, icons, state, actions, tips));
    }
}
pub(crate) const EXTRUDE: &str = "Extrude";
/// A part with no sketch has no profile to extrude, so Extrude asks for a plane to sketch on.
pub(crate) fn extrude_can_start(state: &GuiCommandState) -> bool {
    if state.authoring != crate::GuiAuthoringContext::Part || state.transaction_open {
        return false;
    }
    let Some(browser) = state.browser.as_ref() else {
        return false;
    };
    browser.sketches.is_empty() || state.extrude_available
}

fn add_separator_after(selection: &mut KindSelection, label: &str) {
    let found = selection
        .added
        .iter()
        .position(|entry| matches!(entry, AddedEntry::Command(known) if known == label));
    if let Some(index) = found {
        selection.added.insert(index + 1, AddedEntry::Separator);
    }
}
fn muted_button(ui: &Ui, label: &str) {
    {
        let _muted = ui.begin_disabled(true);
        ui.button(label);
    }
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(NOT_AVAILABLE);
    }
}
fn separator(ui: &Ui) {
    let [x, y] = ui.cursor_screen_pos();
    let height = ui.frame_height();
    let color = ui.style_color(imgui::StyleColor::Text);
    ui.get_window_draw_list()
        .add_line(
            [x + SEPARATOR_WIDTH * 0.5, y + 2.],
            [x + SEPARATOR_WIDTH * 0.5, y + height - 2.],
            color,
        )
        .build();
    ui.dummy([SEPARATOR_WIDTH, height]);
}

#[cfg(test)]
#[path = "../../../tests/application/quick-access-toolbar/selection.rs"]
mod tests;
