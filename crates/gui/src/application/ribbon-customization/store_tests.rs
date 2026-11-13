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
    assert!(customization.draft.changed());
    store.0.borrow_mut().fail_save = false;
    customization
        .relocate_command(RibbonTab::Tools, Command::Sketches, true)
        .unwrap();
    assert!(customization.has_expanded(RibbonTab::Tools));
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
