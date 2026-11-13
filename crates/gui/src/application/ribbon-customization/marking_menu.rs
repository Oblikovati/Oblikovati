// SPDX-License-Identifier: GPL-2.0-only
//! Staged edits for the active sketch marking menu.
use super::RibbonCustomization;
use imgui::Ui;
use oblikovati_api::{
    types::{self, ScreenQuadrant},
    wire::{MarkingMenuItem, MarkingMenuView, SetMarkingMenuArgs},
};

impl RibbonCustomization {
    pub(super) fn sync_marking(&mut self, state: &crate::GuiCommandState) {
        let Some(menus) = &state.marking_menus else {
            return;
        };
        let current = menus.get(types::SKETCH_ENVIRONMENT);
        if self.marking_draft.is_none() || !self.marking_changed() {
            self.marking_draft = Some(current.clone());
            self.marking_base = Some(current);
            self.marking_conflict = false;
            return;
        }
        if self.marking_draft.as_ref() == Some(&current) {
            self.marking_base = Some(current);
            self.marking_conflict = false;
            return;
        }
        self.marking_conflict = self.marking_base.as_ref() != Some(&current);
    }
    pub(super) fn marking_changed(&self) -> bool {
        self.marking_draft
            .as_ref()
            .zip(self.marking_base.as_ref())
            .is_some_and(|(draft, base)| draft != base)
    }
    pub(super) fn submit_marking(&self, actions: &mut Vec<crate::GuiShellAction>) -> bool {
        if !self.marking_changed() || self.marking_conflict {
            return false;
        }
        if let Some(menu) = &self.marking_draft {
            actions.push(crate::GuiShellAction::SetMarkingMenu(SetMarkingMenuArgs {
                menu: menu.clone(),
            }));
            return true;
        }
        false
    }
    pub(super) fn finish_apply(&mut self) {
        if self.close_after_apply && !self.keyboard_changed() && !self.marking_changed() {
            self.open = false;
            self.close_after_apply = false;
        }
    }
    pub(super) fn marking_editor(&mut self, ui: &Ui) {
        let Some(menu) = &mut self.marking_draft else {
            ui.text("Marking menu is unavailable.");
            return;
        };
        ui.text("Environment: 2D Sketch");
        ui.text("Sub-Environment: None Selected");
        for (slot, label) in [
            (0, "North"),
            (1, "North East"),
            (2, "East"),
            (3, "South East"),
            (4, "South"),
            (5, "South West"),
            (6, "West"),
            (7, "North West"),
        ] {
            slot_editor(ui, menu, slot, label);
        }
        if ui.button("Restore Default") {
            *menu = oblikovati_model::marking_menu::MarkingMenus::default()
                .get(types::SKETCH_ENVIRONMENT);
        }
    }
}
fn slot_editor(ui: &Ui, menu: &mut MarkingMenuView, slot: u8, label: &str) {
    let current = menu
        .quadrants
        .iter()
        .flatten()
        .find(|item| item.quadrant.0 == slot)
        .map(|item| item.command_id.as_str())
        .unwrap_or("");
    let preview = crate::sketch::marking_menu::action(current).map_or_else(
        || if current.is_empty() { "None" } else { current },
        |(label, _)| label,
    );
    ui.set_next_item_width(200.);
    let Some(_combo) = ui.begin_combo(format!("{label}##marking-slot-{slot}"), preview) else {
        return;
    };
    if ui.selectable("None") {
        replace_slot(menu, slot, "");
    }
    for (id, name) in crate::sketch::marking_menu::COMMANDS {
        if ui.selectable(name) {
            replace_slot(menu, slot, id);
        }
    }
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
