// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::application_prompts::{
    ApplicationPrompt, PromptFrequency, PromptPreferenceAccess, PromptPreferences,
};
#[test]
fn an_absent_group_shows_every_prompt() {
    let file = load(&disk(b"")).unwrap();
    assert_eq!(file.prompt_preferences(), PromptPreferences::default());
}
#[test]
fn a_saved_frequency_survives_a_new_load_and_keeps_other_groups() {
    let disk = disk(b"futureGroup: {kept: 1}\n");
    let mut file = load(&disk).unwrap();
    let prompts = PromptPreferences::default().with(
        ApplicationPrompt::SaveChanges,
        PromptFrequency::SuppressPermanently,
    );
    file.set_prompt_preferences(prompts).unwrap();
    assert_eq!(load(&disk).unwrap().prompt_preferences(), prompts);
    let encoded = String::from_utf8(disk.borrow().committed.clone()).unwrap();
    assert!(
        encoded.contains("futureGroup"),
        "options {encoded:?}: expected the group kept"
    );
}
#[test]
fn a_session_choice_is_refused_on_save_and_on_load_with_its_name() {
    let disk = disk(b"");
    let mut file = load(&disk).unwrap();
    let session = PromptPreferences::default().with(
        ApplicationPrompt::ExitEnvironment,
        PromptFrequency::SuppressForSession,
    );
    assert!(
        file.set_prompt_preferences(session)
            .unwrap_err()
            .contains("SuppressForSession")
    );
    let stored = super::disk(b"prompts: {save_changes: suppress_for_session}");
    let refusal = load(&stored).err().expect("expected a refused file");
    assert!(refusal.to_string().contains("SuppressForSession"));
}
