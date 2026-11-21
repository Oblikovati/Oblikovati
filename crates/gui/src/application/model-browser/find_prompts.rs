// SPDX-License-Identifier: GPL-2.0-only
//! The small windows the Find dialog opens: grouping, no match, and its saved searches.
use super::find_criteria::{FindCriterion, FindObject};
use crate::application::document_saving::folder_browser::center_dialog;
use imgui::Ui;

pub(super) const GROUP_TITLE: &str = "Group Criteria";
pub(super) const NO_MATCH_TITLE: &str = "Find##no-match";
pub(super) const SAVE_TITLE: &str = "Save Search";
pub(super) const OPEN_TITLE: &str = "Open Search";
const GROUP_MESSAGE: &str = "Criteria using the same property must be grouped.";
const GROUP_QUESTION: &str = "Add this criterion using the OR operator?";
const NO_MATCH_MESSAGE: &str = "No objects matched the search criteria.";
const PROMPT_SIZE: [f32; 2] = [440., 140.];
const PICKER_SIZE: [f32; 2] = [420., 300.];
const PICKER_LIST_HEIGHT: f32 = 170.;
const BUTTON: [f32; 2] = [80., 0.];

/// A search kept by name, in this session.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct SavedSearch {
    pub name: String,
    pub object: FindObject,
    pub match_case: bool,
    pub criteria: Vec<FindCriterion>,
}
/// True for Yes, false for No, while the question stands nothing.
pub(super) fn group_prompt(ui: &Ui) -> Option<bool> {
    center_dialog(ui, PROMPT_SIZE);
    let _prompt = ui.begin_modal_popup(GROUP_TITLE)?;
    ui.text(GROUP_MESSAGE);
    ui.text(GROUP_QUESTION);
    let yes = crate::widget_abi::button(ui, "Yes", BUTTON);
    ui.same_line();
    let no = crate::widget_abi::button(ui, "No", BUTTON);
    if yes || no {
        ui.close_current_popup();
        return Some(yes);
    }
    None
}
pub(super) fn no_match_message(ui: &Ui) {
    center_dialog(ui, PROMPT_SIZE);
    let Some(_message) = ui.begin_modal_popup(NO_MATCH_TITLE) else {
        return;
    };
    ui.text(NO_MATCH_MESSAGE);
    if crate::widget_abi::button(ui, "OK", BUTTON) {
        ui.close_current_popup();
    }
    ui.same_line();
    let _muted = ui.begin_disabled(true);
    crate::widget_abi::button(ui, "Help", BUTTON);
}
/// The name typed, when OK takes it; a name is what makes it possible to find the search again.
pub(super) fn save_prompt(ui: &Ui, name: &mut String) -> Option<String> {
    center_dialog(ui, PROMPT_SIZE);
    let _prompt = ui.begin_modal_popup(SAVE_TITLE)?;
    if ui.is_window_appearing() {
        ui.set_keyboard_focus_here();
    }
    ui.input_text("Search name", name).build();
    let chosen = {
        let _muted = ui.begin_disabled(name.trim().is_empty());
        crate::widget_abi::button(ui, "OK", BUTTON)
    };
    ui.same_line();
    if crate::widget_abi::button(ui, "Cancel", BUTTON) {
        ui.close_current_popup();
    }
    if chosen {
        ui.close_current_popup();
        return Some(name.trim().to_owned());
    }
    None
}
/// What the picker asks of the saved search it has chosen.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum PickerAnswer {
    Open(usize),
    Delete(usize),
}
pub(super) fn open_picker(
    ui: &Ui,
    saved: &[SavedSearch],
    chosen: &mut usize,
) -> Option<PickerAnswer> {
    center_dialog(ui, PICKER_SIZE);
    let _picker = ui.begin_modal_popup(OPEN_TITLE)?;
    ui.child_window("##saved-searches")
        .size([0., PICKER_LIST_HEIGHT])
        .border(true)
        .build(|| {
            for (index, search) in saved.iter().enumerate() {
                if crate::widget_abi::dialog_selectable(ui, &search.name, index == *chosen) {
                    *chosen = index;
                }
            }
        });
    let (open, delete) = {
        let _muted = ui.begin_disabled(saved.is_empty());
        let open = crate::widget_abi::button(ui, "Open", BUTTON);
        ui.same_line();
        let delete = crate::widget_abi::button(ui, "Delete", BUTTON);
        ui.same_line();
        (open, delete)
    };
    if crate::widget_abi::button(ui, "Cancel", BUTTON) {
        ui.close_current_popup();
    }
    if delete {
        return Some(PickerAnswer::Delete(*chosen));
    }
    if open {
        ui.close_current_popup();
        return Some(PickerAnswer::Open(*chosen));
    }
    None
}
