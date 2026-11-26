// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;
#[test]
fn surviving_window_retains_its_identity_across_close_and_reopen() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    owner.new_document_window().unwrap();
    let extra = owner.active_view_slot();
    owner.close_document_window(extra).unwrap();
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .document_windows
            .is_empty()
    );

    // No GUI state is sampled between creating a view and closing the original.
    owner.new_document_window().unwrap();
    let first = owner.active_view_slot();
    owner.close_document_window(document).unwrap();
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.document_windows[0].id, first);
    assert_eq!(state.active_window, Some(first));
    owner.new_document_window().unwrap();
    let second = owner.active_view_slot();
    owner.close_document_window(first).unwrap();
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.document_windows[0].id, second);
    assert_eq!(state.active_window, Some(second));
    owner
        .close_document_from_gui(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    assert!(owner.document_windows.is_empty());
}
#[test]
fn closing_an_inactive_window_keeps_the_active_window_and_sketch() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    owner.new_document_window().unwrap();
    owner.new_document_window().unwrap();
    let active = owner.active_view_slot();
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.close_document_window(document).unwrap();
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.active_window, Some(active));
    assert_eq!(state.document, Some(document));
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
}
#[test]
fn closing_an_inactive_original_window_keeps_the_other_document_active() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    owner.new_document_window().unwrap();
    let survivor = owner.active_view_slot();
    let second = create(&mut owner, &awake);
    owner.close_document_window(first).unwrap();
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.document, Some(second));
    assert_eq!(state.active_window, Some(second));
    assert_eq!(state.document_windows[0].id, survivor);
    owner
        .activate_document_from_gui(&ActivateDocumentArgs { id: first })
        .unwrap();
    assert_eq!(
        owner.gui_command_state().unwrap().active_window,
        Some(survivor)
    );
}
#[test]
fn document_snapshot_and_gui_activation_follow_the_same_file_owner() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    let second = create(&mut owner, &awake);
    let rows = owner.gui_command_state().unwrap().documents;
    assert_eq!(
        rows.iter().map(|row| row.id).collect::<Vec<_>>(),
        [first, second]
    );
    assert!(rows[0].visible && rows[1].visible && rows[1].active);
    let request = ActivateDocumentArgs { id: first };
    assert!(owner.activate_document_from_gui(&request).unwrap());
    assert!(!owner.activate_document_from_gui(&request).unwrap());
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.document, Some(first));
    assert!(state.documents[0].active && !state.documents[1].active);
    assert!(
        owner
            .activate_document_from_gui(&ActivateDocumentArgs { id: 999 })
            .is_err()
    );
    assert_eq!(owner.gui_command_state().unwrap(), state);
}
#[test]
fn a_sketch_edit_parks_with_its_document_and_resumes_on_return() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    let second = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(second, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    let away = ActivateDocumentArgs { id: first };
    assert!(owner.activate_document_from_gui(&away).unwrap());
    assert!(owner.active_sketch_edit().is_none());
    let back = ActivateDocumentArgs { id: second };
    assert!(owner.activate_document_from_gui(&back).unwrap());
    let resumed = owner.active_sketch_edit().unwrap();
    assert_eq!((resumed.document, resumed.sketch), (second, sketch));
}
#[test]
fn closing_a_document_forgets_its_parked_edit() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    let second = create(&mut owner, &awake);
    owner
        .create_sketch_for_edit(second, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    let away = ActivateDocumentArgs { id: first };
    owner.activate_document_from_gui(&away).unwrap();
    assert_eq!(owner.parked_sketch_edits.len(), 1);
    owner
        .close_document_from_gui(&oblikovati_api::wire::CloseDocumentArgs {
            id: second,
            force: true,
        })
        .unwrap();
    assert!(owner.parked_sketch_edits.is_empty());
}
#[test]
fn gui_activation_cannot_abandon_an_open_transaction() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    let second = create(&mut owner, &awake);
    let request = ActivateDocumentArgs { id: first };
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Edit".into(),
        })
        .unwrap();
    assert!(owner.activate_document_from_gui(&request).is_err());
    assert_eq!(owner.gui_command_state().unwrap().document, Some(second));
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
    assert!(owner.activate_document_from_gui(&request).unwrap());
}
#[test]
fn gui_document_switch_refuses_hidden_targets_and_target_transactions() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    let second = create(&mut owner, &awake);
    owner.files.set_visible(first, false).unwrap();
    assert!(!owner.gui_command_state().unwrap().documents[0].visible);
    assert!(
        owner
            .activate_document_from_gui(&ActivateDocumentArgs { id: first })
            .is_err()
    );
    owner.files.set_visible(first, true).unwrap();
    owner
        .activate_document_from_gui(&ActivateDocumentArgs { id: first })
        .unwrap();
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Held edit".into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: second })
        .unwrap();
    assert!(
        owner
            .activate_document_from_gui(&ActivateDocumentArgs { id: first })
            .is_err()
    );
    assert_eq!(owner.gui_command_state().unwrap().document, Some(second));
}

#[test]
fn switching_documents_retains_each_selected_camera_slot() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    owner.new_document_window().unwrap();
    let first_view = owner.active_view_slot();
    let second = create(&mut owner, &awake);
    owner.new_document_window().unwrap();
    let second_view = owner.active_view_slot();

    assert_eq!(owner.camera_slots()[&first], first_view);
    assert_eq!(owner.camera_slots()[&second], second_view);
    owner
        .activate_document_from_gui(&ActivateDocumentArgs { id: first })
        .unwrap();
    assert_eq!(owner.active_view_slot(), first_view);
    owner.close_document_window(second_view).unwrap();
    assert_eq!(owner.active_view_slot(), first_view);
    assert_eq!(owner.camera_slots().get(&second), None);
    owner
        .activate_document_from_gui(&ActivateDocumentArgs { id: second })
        .unwrap();
    assert_eq!(owner.active_view_slot(), second);
}
