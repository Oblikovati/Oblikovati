// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;
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
fn gui_activation_cannot_abandon_a_sketch_or_an_open_transaction() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    let second = create(&mut owner, &awake);
    owner
        .create_sketch_for_edit(second, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    let request = ActivateDocumentArgs { id: first };
    assert!(owner.activate_document_from_gui(&request).is_err());
    assert_eq!(owner.active_sketch_edit().unwrap().document, second);
    owner.finish_sketch_edit().unwrap();
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
