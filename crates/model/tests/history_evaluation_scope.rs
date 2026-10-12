// SPDX-License-Identifier: GPL-2.0-only
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
use oblikovati_model::{
    history::{DocumentEditHistory, HistoryError},
    identity::RecipeIdAllocator,
};
fn history() -> DocumentEditHistory {
    let document = fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    DocumentEditHistory::new(document, ids).unwrap()
}
#[test]
fn refresh_invalidates_the_captured_generation_without_changing_its_inputs() {
    let mut history = history();
    let captured = history.evaluation_snapshot();
    assert!(captured.scope().is_current(&history));
    assert!(captured.scope().belongs_to(&history));
    history
        .refresh(|snapshot| snapshot.parameters.set_expression(3, "3 cm"))
        .unwrap();
    assert!(!captured.scope().is_current(&history));
    assert!(captured.scope().is_superseded());
    assert_eq!(
        captured.snapshot().parameters.model_value(3).unwrap().value,
        2.
    );
    assert_eq!(captured.scope().generation(), 0);
}
#[test]
fn closed_history_cancels_capture_even_when_a_transaction_token_survives() {
    let mut owner = history();
    let token = owner.begin("Open").unwrap();
    let captured = owner.evaluation_snapshot();
    drop(owner);
    assert!(captured.scope().is_superseded());
    assert!(!captured.scope().belongs_to(&history()));
    drop(token);
}
#[test]
fn an_owner_edit_is_visible_to_a_waiting_worker_without_sharing_mutable_recipe() {
    let mut owner = history();
    let captured = owner.evaluation_snapshot();
    let (ready, wait_ready) = std::sync::mpsc::channel();
    let (proceed, wait_proceed) = std::sync::mpsc::channel();
    let worker = std::thread::spawn(move || {
        ready.send(()).unwrap();
        wait_proceed.recv().unwrap();
        assert!(captured.scope().is_superseded());
        assert_eq!(
            captured.snapshot().parameters.model_value(3).unwrap().value,
            2.
        );
    });
    wait_ready.recv().unwrap();
    commit_depth_edit(&mut owner);
    proceed.send(()).unwrap();
    worker.join().unwrap();
}
#[test]
fn a_rejected_candidate_preserves_the_current_evaluation_scope() {
    let mut owner = history();
    let captured = owner.evaluation_snapshot();
    let token = owner.begin("Rejected").unwrap();
    let result = owner.edit(&token, |_, _| Err::<(), _>(HistoryError::Empty("refused")));
    assert!(result.is_err());
    assert!(captured.scope().is_current(&owner));
    owner.discard_unchanged(&token).unwrap();
    assert!(captured.scope().is_current(&owner));
}

fn commit_depth_edit(owner: &mut DocumentEditHistory) {
    let token = owner.begin("Depth").unwrap();
    owner
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    owner.commit(&token, false).unwrap();
}
