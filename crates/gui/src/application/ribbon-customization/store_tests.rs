// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::{cell::RefCell, rc::Rc};

#[derive(Default)]
struct StoredSettings {
    bytes: Option<Vec<u8>>,
    fail_save: bool,
}
#[derive(Clone, Default)]
struct FakeRibbonStore(Rc<RefCell<StoredSettings>>);
impl GuiRibbonSettingsStore for FakeRibbonStore {
    fn load(&mut self) -> Result<Option<Vec<u8>>, String> {
        Ok(self.0.borrow().bytes.clone())
    }
    fn save(&mut self, bytes: &[u8]) -> Result<(), String> {
        let mut state = self.0.borrow_mut();
        if state.fail_save {
            return Err("ribbon settings: expected writable storage".into());
        }
        state.bytes = Some(bytes.to_vec());
        Ok(())
    }
}
#[test]
fn plane_group_and_placement_persist_independently_and_roll_back_failed_saves() {
    let store = FakeRibbonStore::default();
    let id = "part.model.work-features.plane";
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    customization.relocate_builtin(id, true).unwrap();
    customization.set_builtin_ungrouped(id, true).unwrap();
    store.0.borrow_mut().fail_save = true;
    assert!(customization.set_builtin_ungrouped(id, false).is_err());
    assert!(customization.builtin_ungrouped(id));
    store.0.borrow_mut().fail_save = false;
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store.clone())).unwrap();
    assert!(reopened.builtin_expanded(id));
    assert!(reopened.builtin_ungrouped(id));
    reopened.set_builtin_ungrouped(id, false).unwrap();
    reopened.relocate_builtin(id, false).unwrap();
    let mut restored = RibbonCustomization::default();
    restored.set_store(Box::new(store)).unwrap();
    assert!(!restored.builtin_expanded(id));
    assert!(!restored.builtin_ungrouped(id));
}
#[test]
fn shell_button_size_keeps_its_saved_value_through_appearance_and_failed_save() {
    let store = FakeRibbonStore::default();
    let id = crate::shell_ribbon::ID;
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    assert!(!customization.builtin_large(id));
    customization.set_builtin_large(id, true).unwrap();
    customization.select_appearance(crate::ribbon_command_layout::RibbonAppearance::TextOff);
    store.0.borrow_mut().fail_save = true;
    assert!(customization.set_builtin_large(id, false).is_err());
    assert!(customization.builtin_large(id));
    store.0.borrow_mut().fail_save = false;
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store.clone())).unwrap();
    assert!(reopened.builtin_large(id));
    assert_eq!(
        reopened.appearance(),
        crate::ribbon_command_layout::RibbonAppearance::TextOff
    );
    reopened.reset().unwrap();
    let mut restored = RibbonCustomization::default();
    restored.set_store(Box::new(store)).unwrap();
    assert!(!restored.builtin_large(id));
}
#[test]
fn applied_settings_reload_and_failed_saves_leave_the_draft_pending() {
    let store = FakeRibbonStore::default();
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    customization.open(RibbonTab::Tools);
    customization
        .draft
        .reset_source_selection(Command::Sketches);
    customization.draft.add();
    store.0.borrow_mut().fail_save = true;
    assert!(customization.apply().is_err());
    assert!(customization.draft.changed());
    assert!(!customization.has_panel(RibbonTab::Tools));
    assert!(store.0.borrow().bytes.is_none());
    store.0.borrow_mut().fail_save = false;
    customization.apply().unwrap();
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store)).unwrap();
    assert!(reopened.has_panel(RibbonTab::Tools));
    assert!(!reopened.draft.changed());
}
#[test]
fn corrupt_saved_settings_are_rejected_without_a_partial_store() {
    let store = FakeRibbonStore::default();
    store.0.borrow_mut().bytes = Some(b"<invalid".to_vec());
    let mut customization = RibbonCustomization::default();
    assert!(customization.set_store(Box::new(store)).is_err());
    assert!(customization.store.is_none());
    assert!(!customization.has_panel(RibbonTab::Tools));
}

#[test]
fn reset_saves_defaults_before_changing_applied_ribbon() {
    let store = FakeRibbonStore::default();
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    customization.open(RibbonTab::Tools);
    customization
        .draft
        .reset_source_selection(Command::Sketches);
    customization.draft.add();
    customization.apply().unwrap();
    let saved = store.0.borrow().bytes.clone();
    store.0.borrow_mut().fail_save = true;
    assert!(customization.reset().is_err());
    assert!(customization.has_panel(RibbonTab::Tools));
    assert_eq!(store.0.borrow().bytes, saved);
    store.0.borrow_mut().fail_save = false;
    customization.reset().unwrap();
    assert!(!customization.has_panel(RibbonTab::Tools));
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store)).unwrap();
    assert!(!reopened.has_panel(RibbonTab::Tools));
}

#[test]
fn moving_a_command_saves_atomically_without_committing_other_drafts() {
    let store = FakeRibbonStore::default();
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    customization.open(RibbonTab::Tools);
    customization
        .draft
        .reset_source_selection(Command::Sketches);
    customization.draft.add();
    customization.apply().unwrap();
    customization
        .draft
        .reset_source_selection(Command::Tool(crate::GuiSketchTool::Line));
    customization.draft.add();
    store.0.borrow_mut().fail_save = true;
    assert!(
        customization
            .relocate_command(RibbonTab::Tools, Command::Sketches, true)
            .is_err()
    );
    assert!(!customization.has_expanded(RibbonTab::Tools));
    assert_eq!(customization.expanded_command_count(RibbonTab::Tools), 0);
    assert!(customization.draft.changed());
    store.0.borrow_mut().fail_save = false;
    customization
        .relocate_command(RibbonTab::Tools, Command::Sketches, true)
        .unwrap();
    assert!(customization.has_expanded(RibbonTab::Tools));
    assert_eq!(customization.expanded_command_count(RibbonTab::Tools), 1);
    assert!(customization.draft.changed());
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store)).unwrap();
    assert!(reopened.has_expanded(RibbonTab::Tools));
    assert_eq!(reopened.draft.applied(RibbonTab::Tools).len(), 1);
}

#[test]
fn built_in_relocation_is_atomic_and_survives_reload_and_reset() {
    let store = FakeRibbonStore::default();
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    let id = "part.tools.clipboard.copy";
    customization.open(RibbonTab::Tools);
    customization
        .draft
        .reset_source_selection(Command::Sketches);
    customization.draft.add();
    store.0.borrow_mut().fail_save = true;
    assert!(customization.relocate_builtin(id, true).is_err());
    assert!(!customization.builtin_expanded(id));
    store.0.borrow_mut().fail_save = false;
    customization.relocate_builtin(id, true).unwrap();
    assert!(customization.draft.changed());
    assert!(!customization.has_panel(RibbonTab::Tools));
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store.clone())).unwrap();
    assert!(reopened.builtin_expanded(id));
    reopened.reset().unwrap();
    let mut reset = RibbonCustomization::default();
    reset.set_store(Box::new(store)).unwrap();
    assert!(!reset.builtin_expanded(id));
}

#[test]
fn appearance_is_saved_atomically_and_reloaded_with_other_ribbon_settings() {
    use crate::ribbon_command_layout::RibbonAppearance;
    let store = FakeRibbonStore::default();
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    customization.staged_appearance = RibbonAppearance::Compact;
    store.0.borrow_mut().fail_save = true;
    assert!(customization.apply().is_err());
    assert_eq!(customization.appearance, RibbonAppearance::Normal);
    store.0.borrow_mut().fail_save = false;
    customization.apply().unwrap();
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store)).unwrap();
    assert_eq!(reopened.appearance, RibbonAppearance::Compact);
    reopened.reset().unwrap();
    assert_eq!(reopened.appearance, RibbonAppearance::Normal);
}

#[test]
fn selecting_an_appearance_keeps_applied_state_when_saving_fails() {
    use crate::ribbon_command_layout::RibbonAppearance;
    let store = FakeRibbonStore::default();
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    store.0.borrow_mut().fail_save = true;
    customization.select_appearance(RibbonAppearance::Small);
    assert_eq!(customization.appearance(), RibbonAppearance::Normal);
    assert!(customization.command_error.is_some());
    store.0.borrow_mut().fail_save = false;
    customization.select_appearance(RibbonAppearance::Compact);
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store)).unwrap();
    assert_eq!(reopened.appearance(), RibbonAppearance::Compact);
}

#[test]
fn move_body_main_placement_round_trips_and_reset_restores_additional_area() {
    let store = FakeRibbonStore::default();
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    assert!(!crate::move_body_ribbon::is_main(&customization));
    crate::move_body_ribbon::set_main(&mut customization, true).unwrap();
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store.clone())).unwrap();
    assert!(crate::move_body_ribbon::is_main(&reopened));
    crate::move_body_ribbon::set_main(&mut reopened, false).unwrap();
    let mut restored = RibbonCustomization::default();
    restored.set_store(Box::new(store)).unwrap();
    assert!(!crate::move_body_ribbon::is_main(&restored));
}

#[test]
fn shell_placement_persists_and_restores_without_moving_other_commands() {
    let store = FakeRibbonStore::default();
    let mut customization = RibbonCustomization::default();
    customization.set_store(Box::new(store.clone())).unwrap();
    assert!(!crate::shell_ribbon::is_expanded(&customization));
    customization
        .relocate_builtin(crate::shell_ribbon::ID, true)
        .unwrap();
    let mut reopened = RibbonCustomization::default();
    reopened.set_store(Box::new(store.clone())).unwrap();
    assert!(crate::shell_ribbon::is_expanded(&reopened));
    assert!(!crate::move_body_ribbon::is_main(&reopened));
    reopened
        .relocate_builtin(crate::shell_ribbon::ID, false)
        .unwrap();
    let mut restored = RibbonCustomization::default();
    restored.set_store(Box::new(store)).unwrap();
    assert!(!crate::shell_ribbon::is_expanded(&restored));
}
