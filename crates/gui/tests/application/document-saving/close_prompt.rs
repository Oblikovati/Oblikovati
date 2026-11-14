// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::GuiDocumentStorage;
use oblikovati_model::application_prompts::PromptPreferences;
impl ClosePrompt {
    fn is_waiting(&self) -> bool {
        !self.asking.is_empty() || !self.leaving.is_empty()
    }
}
struct Row {
    id: u64,
    dirty: bool,
    edited: bool,
    path: Option<&'static str>,
}
fn state(rows: &[Row]) -> GuiCommandState {
    GuiCommandState {
        document: rows.first().map(|row| row.id),
        documents: rows
            .iter()
            .map(|row| DocumentInfo {
                id: row.id,
                name: format!("Part{}", row.id),
                dirty: row.dirty,
                visible: true,
                ..Default::default()
            })
            .collect(),
        document_storage: rows
            .iter()
            .map(|row| {
                let storage = GuiDocumentStorage {
                    path: row.path.map(str::to_string),
                    unsaved_edits: row.edited,
                };
                (row.id, storage)
            })
            .collect(),
        ..Default::default()
    }
}
const EDITED_NEW: fn(u64) -> Row = |id| Row {
    id,
    dirty: true,
    edited: true,
    path: None,
};
const EDITED_SAVED: fn(u64) -> Row = |id| Row {
    id,
    dirty: true,
    edited: true,
    path: Some("a.opd"),
};
const UNTOUCHED_NEW: fn(u64) -> Row = |id| Row {
    id,
    dirty: true,
    edited: false,
    path: None,
};
const CLEAN_SAVED: fn(u64) -> Row = |id| Row {
    id,
    dirty: false,
    edited: false,
    path: Some("a.opd"),
};

#[test]
fn an_edited_document_waits_and_the_other_commands_proceed() {
    let state = state(&[EDITED_NEW(1), CLEAN_SAVED(2)]);
    let mut prompt = ClosePrompt::default();
    let mut actions = vec![
        close(1, false),
        close(2, false),
        close(1, true),
        GuiShellAction::Save,
    ];
    prompt.intercept(&state, &mut actions);
    assert_eq!(
        actions,
        [close(2, false), close(1, true), GuiShellAction::Save]
    );
    assert_eq!(prompt.asking, [1]);
    assert!(prompt.is_waiting() && prompt.needs_frame());
}
#[test]
fn an_untouched_new_part_closes_forced_without_a_prompt() {
    let state = state(&[UNTOUCHED_NEW(1)]);
    let mut prompt = ClosePrompt::default();
    let mut actions = vec![close(1, false)];
    prompt.intercept(&state, &mut actions);
    assert_eq!(actions, [close(1, true)]);
    assert!(!prompt.is_waiting());
}
#[test]
fn saving_a_document_with_a_file_activates_saves_and_closes_unforced() {
    let state = state(&[CLEAN_SAVED(1), EDITED_SAVED(2)]);
    let mut outcome = CloseOutcome::default();
    answer(&state, 2, CloseAnswer::Save, &mut outcome);
    assert_eq!(
        outcome.actions,
        [
            activate_unless_active(&state, 2).unwrap(),
            GuiShellAction::Save,
            close(2, false),
        ]
    );
    assert!(outcome.first_saves.is_empty());
}
#[test]
fn saving_a_document_without_a_file_asks_for_the_first_save() {
    let state = state(&[EDITED_NEW(1)]);
    let mut outcome = CloseOutcome::default();
    answer(&state, 1, CloseAnswer::Save, &mut outcome);
    assert_eq!(outcome.first_saves, [1]);
    assert!(outcome.actions.is_empty());
}
#[test]
fn discarding_forces_the_close_and_keeping_does_nothing() {
    let state = state(&[EDITED_NEW(1)]);
    let mut outcome = CloseOutcome::default();
    answer(&state, 1, CloseAnswer::Keep, &mut outcome);
    assert_eq!(outcome, CloseOutcome::default());
    answer(&state, 1, CloseAnswer::Discard, &mut outcome);
    assert_eq!(outcome.actions, [close(1, true)]);
}
#[test]
fn a_suppressed_prompt_saves_and_never_discards() {
    let mut state = state(&[EDITED_SAVED(1)]);
    state.prompts = PromptPreferences::default().with(
        ApplicationPrompt::SaveChanges,
        PromptFrequency::SuppressPermanently,
    );
    let mut prompt = ClosePrompt::default();
    let mut actions = vec![close(1, false)];
    let outcome = prompt.intercept(&state, &mut actions);
    assert!(actions.is_empty() && !prompt.is_waiting());
    assert_eq!(outcome.actions, [GuiShellAction::Save, close(1, false)]);
}
#[test]
fn once_per_operation_applies_one_answer_to_every_waiting_document() {
    let mut state = state(&[EDITED_SAVED(1), EDITED_SAVED(2)]);
    state.prompts = PromptPreferences::default().with(
        ApplicationPrompt::SaveChanges,
        PromptFrequency::OncePerOperation,
    );
    let mut prompt = ClosePrompt {
        asking: vec![1, 2],
        ..Default::default()
    };
    let mut outcome = CloseOutcome::default();
    prompt.resolve(&state, CloseAnswer::Discard, &mut outcome);
    assert_eq!(outcome.actions, [close(1, true), close(2, true)]);
    assert!(!prompt.is_waiting());
}
#[test]
fn always_show_asks_again_for_the_next_document() {
    let state = state(&[EDITED_SAVED(1), EDITED_SAVED(2)]);
    let mut prompt = ClosePrompt {
        asking: vec![1, 2],
        ..Default::default()
    };
    let mut outcome = CloseOutcome::default();
    prompt.resolve(&state, CloseAnswer::Discard, &mut outcome);
    assert_eq!(outcome.actions, [close(1, true)]);
    assert_eq!(prompt.asking, [2]);
}
#[test]
fn cancel_keeps_every_waiting_document() {
    let state = state(&[EDITED_NEW(1), EDITED_NEW(2)]);
    let mut prompt = ClosePrompt {
        asking: vec![1, 2],
        ..Default::default()
    };
    let mut outcome = CloseOutcome::default();
    prompt.resolve(&state, CloseAnswer::Keep, &mut outcome);
    assert_eq!(outcome, CloseOutcome::default());
    assert!(!prompt.is_waiting());
}
#[test]
fn closing_during_a_sketch_edit_asks_to_leave_first() {
    let mut state = state(&[EDITED_NEW(1)]);
    state.authoring = GuiAuthoringContext::Sketch;
    let mut prompt = ClosePrompt::default();
    let mut actions = vec![close(1, false)];
    prompt.intercept(&state, &mut actions);
    assert!(actions.is_empty());
    assert_eq!(prompt.leaving, [1]);
    assert!(prompt.asking.is_empty());
}
#[test]
fn a_clean_inactive_document_closes_during_a_sketch_edit_without_a_prompt() {
    let mut state = state(&[EDITED_NEW(1), CLEAN_SAVED(2)]);
    state.authoring = GuiAuthoringContext::Sketch;
    assert_eq!(close_route(&state, 2), CloseRoute::Pass);
}
#[test]
fn leaving_finishes_the_sketch_once_and_resumes_the_close_after_it() {
    let mut state = state(&[EDITED_NEW(1), EDITED_NEW(2)]);
    state.authoring = GuiAuthoringContext::Sketch;
    let mut prompt = ClosePrompt::default();
    let mut outcome = CloseOutcome::default();
    prompt.leave_sketch([1], &mut outcome);
    prompt.leave_sketch([2], &mut outcome);
    assert_eq!(outcome.actions, [GuiShellAction::FinishSketch]);
    assert!(prompt.needs_frame());
    state.authoring = GuiAuthoringContext::Part;
    let mut actions = Vec::new();
    prompt.intercept(&state, &mut actions);
    assert_eq!(prompt.asking, [1, 2]);
    assert!(prompt.continuing.is_empty());
}
#[test]
fn a_refused_finish_keeps_the_held_document_open() {
    let mut state = state(&[EDITED_NEW(1)]);
    state.authoring = GuiAuthoringContext::Sketch;
    let mut prompt = ClosePrompt {
        continuing: vec![1],
        ..Default::default()
    };
    let mut actions = Vec::new();
    prompt.intercept(&state, &mut actions);
    assert!(actions.is_empty() && !prompt.is_waiting() && !prompt.needs_frame());
}
#[test]
fn a_suppressed_exit_prompt_finishes_the_sketch_at_once() {
    let mut state = state(&[EDITED_NEW(1)]);
    state.authoring = GuiAuthoringContext::Sketch;
    state.prompts = PromptPreferences::default().with(
        ApplicationPrompt::ExitEnvironment,
        PromptFrequency::SuppressForSession,
    );
    let mut prompt = ClosePrompt::default();
    let mut actions = vec![close(1, false)];
    let outcome = prompt.intercept(&state, &mut actions);
    assert_eq!(outcome.actions, [GuiShellAction::FinishSketch]);
    assert_eq!(prompt.continuing, [1]);
}
#[test]
fn every_frequency_has_its_own_label() {
    let labels: std::collections::BTreeSet<&str> = PromptFrequency::ALL
        .into_iter()
        .map(frequency_label)
        .collect();
    assert_eq!(labels.len(), PromptFrequency::ALL.len());
}
