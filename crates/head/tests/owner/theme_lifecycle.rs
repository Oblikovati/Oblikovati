// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::theme::{ThemeDefinition, ThemeLibrary, ThemeSelectionStore};
use std::{cell::RefCell, collections::BTreeMap, rc::Rc};
#[derive(Default)]
struct LifecycleRecords {
    sources: BTreeMap<String, ThemeDefinition>,
    selected: String,
    fail: Option<&'static str>,
}
struct RecordedLifecycleStore(Rc<RefCell<LifecycleRecords>>);
impl ThemeSelectionStore for RecordedLifecycleStore {
    fn save_selection(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        let mut records = self.0.borrow_mut();
        if records.fail == Some("selection") {
            return Err("injected selection failure".into());
        }
        records.selected = theme.name().into();
        Ok(())
    }
    fn can_create_custom(&self) -> bool {
        true
    }
    fn can_save_custom(&self, theme: &ThemeDefinition) -> bool {
        self.0.borrow().sources.contains_key(theme.name())
    }
    fn create_custom_theme(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        let mut records = self.0.borrow_mut();
        if records.fail == Some("create") {
            return Err("injected creation failure".into());
        }
        assert!(!records.sources.contains_key(theme.name()));
        records.sources.insert(theme.name().into(), theme.clone());
        Ok(())
    }
    fn remove_custom_theme(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        let mut records = self.0.borrow_mut();
        if records.fail == Some("remove") {
            return Err("injected removal failure".into());
        }
        records
            .sources
            .remove(theme.name())
            .ok_or("expected known custom")?;
        Ok(())
    }
}
fn lifecycle_owner() -> (TestOwner, Rc<RefCell<LifecycleRecords>>) {
    let records = Rc::new(RefCell::new(LifecycleRecords::default()));
    let (owner, _) = owner();
    (
        owner.with_theme_store(
            ThemeLibrary::new(Vec::new(), "Dark").unwrap(),
            Box::new(RecordedLifecycleStore(records.clone())),
        ),
        records,
    )
}
fn active_name(owner: &mut TestOwner) -> String {
    Client::new(owner.document_host())
        .theme()
        .active()
        .unwrap()
        .name
}
#[test]
fn duplicate_and_delete_share_api_gui_state_and_leave_document_history_empty() {
    let (mut owner, records) = lifecycle_owner();
    assert!(owner.duplicate_theme_from_gui("Dark", "Workshop").unwrap());
    assert_eq!(active_name(&mut owner), "Workshop");
    assert_eq!(owner.themes.as_ref().unwrap().revision(), 2);
    assert_eq!(records.borrow().selected, "Workshop");
    assert_eq!(records.borrow().sources.len(), 1);
    let state = owner.gui_command_state().unwrap();
    assert!(
        oblikovati_gui::GuiShellAction::DeleteTheme {
            name: "Workshop".into()
        }
        .enabled(&state)
    );
    assert!(owner.delete_theme_from_gui("Workshop").unwrap());
    assert_eq!(active_name(&mut owner), "Dark");
    assert_eq!(records.borrow().selected, "Dark");
    assert_eq!(owner.themes.as_ref().unwrap().revision(), 3);
    assert!(records.borrow().sources.is_empty());
    assert!(owner.documents().workspace().documents().is_empty());
}
#[test]
fn invalid_and_stale_lifecycle_requests_do_not_write_or_change_the_palette() {
    let (mut owner, records) = lifecycle_owner();
    for (base, name) in [("Light", "Workshop"), ("Dark", ""), ("Dark", "Light")] {
        assert!(owner.duplicate_theme_from_gui(base, name).is_err());
    }
    for name in ["Dark", "Missing"] {
        assert!(owner.delete_theme_from_gui(name).is_err());
    }
    assert_eq!(active_name(&mut owner), "Dark");
    assert!(records.borrow().sources.is_empty());
    let (mut empty, _) = super::owner();
    assert!(empty.duplicate_theme_from_gui("Dark", "Workshop").is_err());
    assert!(empty.delete_theme_from_gui("Workshop").is_err());
}
#[test]
fn source_failures_keep_the_previous_palette_and_source_identity() {
    let (mut owner, records) = lifecycle_owner();
    records.borrow_mut().fail = Some("create");
    assert!(
        owner
            .duplicate_theme_from_gui("Dark", "Workshop")
            .unwrap_err()
            .contains("creation failure")
    );
    assert_eq!(active_name(&mut owner), "Dark");
    records.borrow_mut().fail = None;
    owner.duplicate_theme_from_gui("Dark", "Workshop").unwrap();
    records.borrow_mut().fail = Some("remove");
    assert!(
        owner
            .delete_theme_from_gui("Workshop")
            .unwrap_err()
            .contains("removal failure")
    );
    assert_eq!(active_name(&mut owner), "Workshop");
    assert!(records.borrow().sources.contains_key("Workshop"));
}
#[test]
fn preference_failures_report_completed_file_changes_without_losing_gallery_identity() {
    let (mut owner, records) = lifecycle_owner();
    records.borrow_mut().fail = Some("selection");
    assert!(
        owner
            .duplicate_theme_from_gui("Dark", "Workshop")
            .unwrap_err()
            .contains("was created but selection was not saved")
    );
    assert_eq!(active_name(&mut owner), "Dark");
    assert!(records.borrow().sources.contains_key("Workshop"));
    records.borrow_mut().fail = None;
    owner.select_theme_from_gui("Workshop").unwrap();
    records.borrow_mut().fail = Some("selection");
    assert!(
        owner
            .delete_theme_from_gui("Workshop")
            .unwrap_err()
            .contains("was deleted but selection was not saved")
    );
    assert_eq!(active_name(&mut owner), "Dark");
    assert!(records.borrow().sources.is_empty());
}
#[test]
fn memory_only_libraries_support_lifecycle_without_claiming_disk_saves() {
    let (owner, _) = owner();
    let mut owner = owner.with_themes(ThemeLibrary::new(Vec::new(), "Light").unwrap());
    owner.duplicate_theme_from_gui("Light", "Workshop").unwrap();
    owner.delete_theme_from_gui("Workshop").unwrap();
    assert_eq!(active_name(&mut owner), "Dark");
}
