// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::application_prompts::{
    ApplicationPrompt, PromptFrequency, PromptPreferenceAccess, PromptPreferences,
};
use std::{cell::RefCell, rc::Rc};
#[derive(Clone, Default)]
struct RecordedPromptStore {
    stored: Rc<RefCell<PromptPreferences>>,
    refuse: bool,
}
impl PromptPreferenceAccess for RecordedPromptStore {
    fn prompt_preferences(&self) -> PromptPreferences {
        *self.stored.borrow()
    }
    fn set_prompt_preferences(&mut self, prompts: PromptPreferences) -> Result<(), String> {
        if self.refuse {
            return Err(format!("prompts {prompts:?}: expected a writable store"));
        }
        *self.stored.borrow_mut() = prompts;
        Ok(())
    }
}
fn stored_owner(store: RecordedPromptStore) -> TestOwner {
    owner().0.with_prompt_preferences(Box::new(store))
}
#[test]
fn a_stored_frequency_loads_with_the_owner() {
    let store = RecordedPromptStore::default();
    *store.stored.borrow_mut() = PromptPreferences::default().with(
        ApplicationPrompt::SaveChanges,
        PromptFrequency::SuppressPermanently,
    );
    let owner = stored_owner(store);
    assert_eq!(
        owner.prompt_frequencies.save_changes,
        PromptFrequency::SuppressPermanently
    );
}
#[test]
fn a_session_choice_applies_now_and_never_reaches_the_file() {
    let store = RecordedPromptStore::default();
    let mut owner = stored_owner(store.clone());
    assert!(
        owner
            .change_prompt_frequency(
                ApplicationPrompt::SaveChanges,
                PromptFrequency::SuppressForSession,
            )
            .unwrap()
    );
    assert!(owner.prompt_frequencies.save_changes.suppressed());
    assert_eq!(*store.stored.borrow(), PromptPreferences::default());
}
#[test]
fn a_durable_choice_is_stored_without_the_session_value_of_another_prompt() {
    let store = RecordedPromptStore::default();
    let mut owner = stored_owner(store.clone());
    owner
        .change_prompt_frequency(
            ApplicationPrompt::SaveChanges,
            PromptFrequency::SuppressForSession,
        )
        .unwrap();
    owner
        .change_prompt_frequency(
            ApplicationPrompt::ExitEnvironment,
            PromptFrequency::SuppressPermanently,
        )
        .unwrap();
    let stored = *store.stored.borrow();
    assert_eq!(stored.save_changes, PromptFrequency::AlwaysShow);
    assert_eq!(
        stored.exit_environment,
        PromptFrequency::SuppressPermanently
    );
}
#[test]
fn a_refused_store_leaves_the_current_choice_and_reports_why() {
    let store = RecordedPromptStore {
        refuse: true,
        ..Default::default()
    };
    let mut owner = stored_owner(store);
    let refusal = owner
        .change_prompt_frequency(
            ApplicationPrompt::SaveChanges,
            PromptFrequency::SuppressPermanently,
        )
        .unwrap_err();
    assert!(refusal.contains("writable"));
    assert_eq!(owner.prompt_frequencies, PromptPreferences::default());
}
#[test]
fn an_unchanged_choice_reports_no_change() {
    let mut owner = stored_owner(RecordedPromptStore::default());
    assert!(
        !owner
            .change_prompt_frequency(ApplicationPrompt::SaveChanges, PromptFrequency::AlwaysShow)
            .unwrap()
    );
}
#[test]
fn new_document_names_count_per_kind_and_skip_open_names() {
    let (mut owner, awake) = owner();
    assert_eq!(owner.next_new_document_name("Part"), "Part1");
    assert_eq!(owner.next_new_document_name("Part"), "Part2");
    assert_eq!(owner.next_new_document_name("Assembly"), "Assembly1");
    create(&mut owner, &awake);
    assert_eq!(owner.next_new_document_name("Owner part"), "Owner part1");
    let open = owner.gui_document_rows().unwrap()[0].name.clone();
    owner.new_document_counts.clear();
    let taken = open.strip_suffix(' ').unwrap_or(&open);
    assert_ne!(owner.next_new_document_name("Part"), taken);
}
#[test]
fn a_new_part_has_no_work_to_lose_until_it_is_edited() {
    let (mut owner, awake) = owner();
    let id = create(&mut owner, &awake);
    let storage = owner.gui_document_storage().unwrap();
    assert!(storage[&id].path.is_none());
    assert!(!storage[&id].unsaved_edits);
    assert!(owner.documents().workspace().needs_save(id).unwrap());
}
