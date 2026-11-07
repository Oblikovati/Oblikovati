// SPDX-License-Identifier: GPL-2.0-only
//! The customizable command row: which commands each document kind shows, and where the row sits.
#[path = "plane_selection.rs"]
pub(crate) mod plane_selection;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, Ui};
use oblikovati_api::types::DocumentType;
use std::collections::BTreeMap;

/// Commands the customization menu lists, in its order.
const STANDARD: [StandardCommand; 7] = [
    StandardCommand::New,
    StandardCommand::Open,
    StandardCommand::Save,
    StandardCommand::Undo,
    StandardCommand::Redo,
    StandardCommand::Home,
    StandardCommand::Print,
];
const CUSTOMIZE_MENU: &str = "##quick-access-customize";
const SHORTCUT_MENU: &str = "##quick-access-shortcut";
const UNDO_HISTORY: &str = "##undo-history";
const NOT_AVAILABLE: &str = "This command is not available in this build.";
/// Height of the separate row below the ribbon.
pub(crate) const BELOW_ROW_HEIGHT: f32 = 30.;
const SEPARATOR_WIDTH: f32 = 7.;

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
enum StandardCommand {
    New,
    Open,
    Save,
    Undo,
    Redo,
    Home,
    Print,
}
impl StandardCommand {
    fn label(self) -> &'static str {
        match self {
            Self::New => "New",
            Self::Open => "Open",
            Self::Save => "Save",
            Self::Undo => "Undo",
            Self::Redo => "Redo",
            Self::Home => "Home",
            Self::Print => "Print",
        }
    }
    fn default_shown(self) -> bool {
        self != Self::Print
    }
}
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
    pub open_extrude: bool,
}
/// The toolbar's choices for this session; a later scenario may ask for them to persist.
#[derive(Default)]
pub(crate) struct QuickAccessToolbar {
    kinds: BTreeMap<String, KindSelection>,
    pub below_ribbon: bool,
    shortcut_under_menu: Option<String>,
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
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let selection = self.selection(state).clone();
        for command in STANDARD
            .into_iter()
            .filter(|command| selection.shows(*command))
        {
            self.standard_button(ui, command, state, actions);
            ui.same_line();
        }
        for entry in &selection.added {
            self.added_entry(ui, entry, state);
            ui.same_line();
        }
        self.shortcut_menu(ui, state);
        self.customize_control(ui, state);
    }
    fn standard_button(
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
            StandardCommand::Save => crate::chrome::command_button(
                ui,
                label,
                GuiShellAction::Save,
                state,
                [0.; 2],
                actions,
            ),
            StandardCommand::Undo => {
                crate::chrome::command_button(
                    ui,
                    label,
                    GuiShellAction::Undo,
                    state,
                    [0.; 2],
                    actions,
                );
                self.undo_history(ui, state);
            }
            StandardCommand::Redo => crate::chrome::command_button(
                ui,
                label,
                GuiShellAction::Redo,
                state,
                [0.; 2],
                actions,
            ),
            StandardCommand::Print => muted_button(ui, label),
        }
    }
    /// Right-click on Undo reads what it would undo, latest first.
    fn undo_history(&mut self, ui: &Ui, state: &GuiCommandState) {
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
        for label in state.undo_labels.iter().rev() {
            ui.text(label);
        }
    }
    fn added_entry(&mut self, ui: &Ui, entry: &AddedEntry, state: &GuiCommandState) {
        let AddedEntry::Command(label) = entry else {
            separator(ui);
            return;
        };
        if label == EXTRUDE && extrude_can_start(state) {
            if ui.button(label) {
                match crate::extrude_dialog::start_action(state) {
                    GuiShellAction::ChooseSketchPlane => self.requests.choose_plane = true,
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
            if ui.menu_item_config(command.label()).selected(shown).build() {
                self.selection(state).toggle(command);
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
    /// A ribbon command's own menu; example: right-click Extrude on the 3D Model tab.
    pub(crate) fn ribbon_command_menu(&mut self, ui: &Ui, label: &str, state: &GuiCommandState) {
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
            .enabled(!present)
            .build()
        {
            self.selection(state)
                .added
                .push(AddedEntry::Command(label.to_owned()));
        }
    }
    /// Draw the row that takes its own band under the ribbon; example: `toolbar.band(ui, ...)`.
    pub(crate) fn band_below(
        &mut self,
        ui: &Ui,
        position: [f32; 2],
        width: f32,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        crate::widget_abi::window_geometry(
            ui,
            position,
            [width, BELOW_ROW_HEIGHT],
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
            .build(|| self.row(ui, state, actions));
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
