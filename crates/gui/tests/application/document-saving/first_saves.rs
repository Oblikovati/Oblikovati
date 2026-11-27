// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn active(id: u64) -> GuiCommandState {
    GuiCommandState {
        document: Some(id),
        ..Default::default()
    }
}
#[test]
fn the_active_document_opens_its_dialog_at_once() {
    let mut saves = FirstSaves::default();
    saves.extend([7]);
    let mut actions = Vec::new();
    assert_eq!(saves.next(&active(7), &mut actions), Some(7));
    assert!(actions.is_empty() && !saves.is_waiting());
}
#[test]
fn an_inactive_document_activates_first_and_opens_on_the_next_frame() {
    let mut saves = FirstSaves::default();
    saves.extend([7]);
    let mut actions = Vec::new();
    assert_eq!(saves.next(&active(1), &mut actions), None);
    assert_eq!(
        actions,
        [GuiShellAction::ActivateDocument(ActivateDocumentArgs {
            id: 7
        })]
    );
    actions.clear();
    assert_eq!(saves.next(&active(7), &mut actions), Some(7));
    assert!(actions.is_empty());
}
#[test]
fn a_refused_activation_ends_the_operation_and_does_not_spin() {
    let mut saves = FirstSaves::default();
    saves.extend([7, 8]);
    let mut actions = Vec::new();
    assert_eq!(saves.next(&active(1), &mut actions), None);
    assert_eq!(saves.next(&active(1), &mut actions), None);
    assert!(!saves.is_waiting());
    assert_eq!(actions.len(), 1);
}
#[test]
fn cancelling_keeps_the_remaining_documents_open() {
    let mut saves = FirstSaves::default();
    saves.extend([7, 8]);
    saves.cancel();
    assert!(!saves.is_waiting());
    assert_eq!(saves.next(&active(7), &mut Vec::new()), None);
}
