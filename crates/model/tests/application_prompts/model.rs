// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn only_suppressing_choices_quiet_a_prompt() {
    let quiet: Vec<bool> = PromptFrequency::ALL.map(PromptFrequency::suppressed).into();
    assert_eq!(quiet, [false, false, true, true]);
}
#[test]
fn one_prompt_changes_without_the_other() {
    let prompts = PromptPreferences::default().with(
        ApplicationPrompt::ExitEnvironment,
        PromptFrequency::SuppressPermanently,
    );
    assert_eq!(
        prompts.frequency(ApplicationPrompt::ExitEnvironment),
        PromptFrequency::SuppressPermanently
    );
    assert_eq!(
        prompts.frequency(ApplicationPrompt::SaveChanges),
        PromptFrequency::AlwaysShow
    );
}
#[test]
fn a_session_choice_is_refused_as_a_stored_value_with_its_name() {
    let prompts = PromptPreferences::default().with(
        ApplicationPrompt::SaveChanges,
        PromptFrequency::SuppressForSession,
    );
    assert!(
        prompts
            .validate_stored()
            .unwrap_err()
            .contains("SuppressForSession")
    );
    PromptPreferences::default().validate_stored().unwrap();
}
