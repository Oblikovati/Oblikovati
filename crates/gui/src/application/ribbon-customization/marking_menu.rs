// SPDX-License-Identifier: GPL-2.0-only
//! Staged edits for radial and overflow commands.
use super::{RibbonCustomization, marking_environment};
use imgui::Ui;
use oblikovati_api::{
    types::ScreenQuadrant,
    wire::{MarkingMenuItem, MarkingMenuView, SetMarkingMenuArgs},
};
use oblikovati_model::marking_menu::{
    MarkingMenuDisplay, MarkingMenuOverflow, MarkingMenuStyle, MarkingMenus,
};

const STYLES: [(MarkingMenuStyle, &str); 3] = [
    (MarkingMenuStyle::IconAndText, "Show Icon and Text"),
    (MarkingMenuStyle::IconOnly, "Show Icon Only"),
    (MarkingMenuStyle::TextOnly, "Show Text Only"),
];
const OVERFLOW: [(MarkingMenuOverflow, &str); 3] = [
    (MarkingMenuOverflow::Full, "Full Menu"),
    (MarkingMenuOverflow::RadialOnly, "Radial Menu Only"),
    (MarkingMenuOverflow::Short, "Short Menu"),
];
#[path = "marking_search.rs"]
mod marking_search;
const LOCATIONS: [&str; 8] = [
    "North",
    "North East",
    "East",
    "South East",
    "South",
    "South West",
    "West",
    "North West",
];

impl RibbonCustomization {
    pub(super) fn sync_marking(&mut self, state: &crate::GuiCommandState) {
        let Some(menus) = &state.marking_menus else {
            return;
        };
        self.marking_environment
            .get_or_insert_with(|| marking_environment::for_context(state.authoring));
        let display = MarkingMenuDisplay {
            classic: state.sketch_classic_menu,
            ..state.marking_menu_display
        };
        if self.marking_draft.is_none() || !self.marking_changed() {
            self.marking_draft = Some(menus.clone());
            self.marking_base = Some(menus.clone());
            self.marking_display_draft = display;
            self.marking_display_base = display;
            self.marking_conflict = false;
            return;
        }
        if self.marking_draft.as_ref() == Some(menus) && self.marking_display_draft == display {
            self.marking_base = Some(menus.clone());
            self.marking_display_base = display;
            self.marking_conflict = false;
            return;
        }
        self.marking_conflict =
            self.marking_base.as_ref() != Some(menus) || self.marking_display_base != display;
    }
    pub(super) fn marking_changed(&self) -> bool {
        self.marking_draft
            .as_ref()
            .zip(self.marking_base.as_ref())
            .is_some_and(|(draft, base)| {
                draft != base || self.marking_display_draft != self.marking_display_base
            })
    }
    pub(super) fn submit_marking(&self, actions: &mut Vec<crate::GuiShellAction>) -> bool {
        if !self.marking_changed() || self.marking_conflict {
            return false;
        }
        let Some((draft, base)) = self.marking_draft.as_ref().zip(self.marking_base.as_ref())
        else {
            return false;
        };
        let changed: Vec<_> = draft
            .all()
            .into_iter()
            .filter(|menu| base.get(menu.environment) != *menu)
            .collect();
        let display = self.marking_display_draft;
        let action = match (changed.as_slice(), display == self.marking_display_base) {
            ([menu], true) => {
                crate::GuiShellAction::SetMarkingMenu(SetMarkingMenuArgs { menu: menu.clone() })
            }
            _ => crate::GuiShellAction::SetMarkingMenuPreferences(changed, display),
        };
        actions.push(action);
        true
    }
    pub(super) fn finish_apply(&mut self) {
        if self.close_after_apply && !self.keyboard_changed() && !self.marking_changed() {
            self.open.set_open(false);
            self.close_after_apply = false;
        }
    }
    pub(super) fn marking_editor(&mut self, ui: &Ui) {
        let (Some(menus), Some(environment)) =
            (&mut self.marking_draft, &mut self.marking_environment)
        else {
            ui.text("Marking menu is unavailable.");
            return;
        };
        self.marking_selector_open |= marking_environment::selector(ui, environment);
        self.marking_selector_open |=
            marking_environment::sub_environment_selector(ui, *environment);
        let commands = marking_environment::commands(*environment);
        let original = menus.get(*environment);
        let mut menu = original.clone();
        ui.group(|| {
            for (slot, label) in LOCATIONS.into_iter().enumerate() {
                slot_editor(ui, &mut menu, slot as u8, label, &commands);
            }
        });
        ui.same_line();
        ui.group(|| {
            self.marking_selector_open |=
                marking_search::editor(ui, &mut self.marking_search, &mut menu, &commands);
            super::marking_overflow::editor(ui, &mut menu, &commands);
        });
        if ui.button("Restore Default") {
            menu = MarkingMenus::default().get(*environment);
        }
        if menu != original {
            menus
                .set(menu)
                .expect("the editor supplies unique marking-menu locations");
        }
        ui.same_line();
        ui.checkbox(
            "Use Classic Context Menu",
            &mut self.marking_display_draft.classic,
        );
        self.marking_selector_open |= marking_selector(
            ui,
            "Marking Menu Style",
            &mut self.marking_display_draft.style,
            &STYLES,
        );
        self.marking_selector_open |= marking_selector(
            ui,
            "Overflow Menu",
            &mut self.marking_display_draft.overflow,
            &OVERFLOW,
        );
    }
}

fn marking_selector<T: Copy + Eq>(
    ui: &Ui,
    label: &str,
    value: &mut T,
    choices: &[(T, &str)],
) -> bool {
    let Some((_, preview)) = choices.iter().find(|(choice, _)| choice == value) else {
        return false;
    };
    ui.set_next_item_width(190.);
    let Some(_combo) = ui.begin_combo(label, preview) else {
        return false;
    };
    for &(choice, name) in choices {
        if ui
            .selectable_config(name)
            .selected(choice == *value)
            .build()
        {
            *value = choice;
        }
    }
    true
}
fn slot_editor(
    ui: &Ui,
    menu: &mut MarkingMenuView,
    slot: u8,
    label: &str,
    commands: &[(&str, &str)],
) {
    let current = menu
        .quadrants
        .iter()
        .flatten()
        .find(|item| item.quadrant.0 == slot)
        .map(|item| item.command_id.as_str())
        .unwrap_or("");
    let preview = if current.is_empty() {
        "None"
    } else {
        command_label(current, commands)
    };
    ui.set_next_item_width(200.);
    let Some(_combo) = ui.begin_combo(format!("{label}##marking-slot-{slot}"), preview) else {
        return;
    };
    if ui.selectable("None") {
        replace_slot(menu, slot, "");
    }
    for &(id, name) in commands {
        if ui.selectable(name) {
            replace_slot(menu, slot, id);
        }
    }
}
pub(super) fn command_label<'a>(id: &'a str, commands: &[(&'a str, &'a str)]) -> &'a str {
    commands
        .iter()
        .find_map(|&(key, name)| (id == key).then_some(name))
        .unwrap_or(id)
}
fn replace_slot(menu: &mut MarkingMenuView, slot: u8, command: &str) {
    let slots = menu.quadrants.get_or_insert_default();
    slots.retain(|item| item.quadrant.0 != slot);
    if !command.is_empty() {
        slots.push(MarkingMenuItem {
            quadrant: ScreenQuadrant(slot),
            command_id: command.into(),
        });
    }
    slots.sort_by_key(|item| item.quadrant.0);
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::types;
    #[test]
    fn slot_edit_preserves_other_slots_and_stages_a_valid_menu() {
        let mut menu =
            oblikovati_model::marking_menu::MarkingMenus::default().get(types::SKETCH_ENVIRONMENT);
        let original = menu.quadrants.clone().unwrap();
        replace_slot(&mut menu, 0, "Sketch.Rectangle");
        assert_eq!(menu.quadrants.as_ref().unwrap().len(), original.len());
        assert_eq!(
            menu.quadrants.as_ref().unwrap()[0].command_id,
            "Sketch.Rectangle"
        );
        assert!(
            oblikovati_model::marking_menu::MarkingMenus::default()
                .set(menu)
                .is_ok()
        );
    }
}
