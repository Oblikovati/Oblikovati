// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;

#[test]
fn finishing_a_saved_sketch_is_clean_and_undo_reopens_it_for_typed_queries() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &super::sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: document_fixture_path!("/memory/history-sketch.opd").into(),
        })
        .unwrap();
    assert!(!owner.files.workspace().needs_save(document).unwrap());
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    owner.finish_sketch_edit().unwrap();
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(!owner.files.workspace().needs_save(document).unwrap());
    let client = Client::new(owner.document_host());
    client.transactions().undo().unwrap();
    assert!(client.sketch().get(&SketchArgs::default()).unwrap().editing);
    drop(client);
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    owner.poll(32).unwrap();
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    assert!(!owner.files.workspace().needs_save(document).unwrap());
}

#[test]
fn vetoed_finish_keeps_the_active_sketch_and_placement_tool() {
    let (mut owner, _) = super::tool_veto::veto_owner("Finish Sketch");
    owner.create_part_document().unwrap();
    let document = owner.files.workspace().active_document().unwrap();
    let sketch = owner
        .create_sketch_for_edit(document, &super::sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Line)
        .unwrap();
    let history = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    assert!(owner.finish_sketch_edit().unwrap_err().contains("unlocked"));
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    assert!(owner.placement_active());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .history_entries(),
        history
    );
}

#[test]
fn typed_client_observes_each_sketch_history_transition_without_a_desktop_poll() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    owner
        .create_sketch_for_edit(document, &super::sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let client = Client::new(owner.document_host());
    client.transactions().undo().unwrap();
    assert!(client.sketch().get(&SketchArgs::default()).unwrap().editing);
    client.transactions().undo().unwrap();
    assert!(
        client
            .sketch()
            .list()
            .unwrap()
            .sketches
            .unwrap_or_default()
            .is_empty()
    );
    client.transactions().redo().unwrap();
    assert!(client.sketch().get(&SketchArgs::default()).unwrap().editing);
    client.transactions().redo().unwrap();
    assert!(!client.sketch().get(&SketchArgs::default()).unwrap().editing);
    drop(client);
    owner.poll(32).unwrap();
    assert!(owner.active_sketch_edit().is_none());
}

#[test]
fn redo_creation_restores_an_editor_when_the_client_predates_undo() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &super::sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    let client = Client::new(owner.document_host());
    assert!(client.sketch().get(&SketchArgs::default()).unwrap().editing);
    client.transactions().undo().unwrap();
    client.transactions().redo().unwrap();
    assert!(client.sketch().get(&SketchArgs::default()).unwrap().editing);
    drop(client);
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    owner.poll(32).unwrap();
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
}

#[test]
fn a_document_switch_discards_unapplied_sketch_navigation() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    owner
        .create_sketch_for_edit(first, &super::sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let client = Client::new(owner.document_host());
    client.transactions().undo().unwrap();
    client
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Other".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    assert!(!client.sketch().get(&SketchArgs::default()).unwrap().editing);
    drop(client);
    owner.poll(32).unwrap();
    assert!(owner.active_sketch_edit().is_none());
}

#[test]
fn history_jump_coalesces_mode_transitions_without_losing_intermediate_targets() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    owner
        .create_sketch_for_edit(document, &super::sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let client = Client::new(owner.document_host());
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: 0,
        })
        .unwrap();
    assert!(
        client
            .sketch()
            .list()
            .unwrap()
            .sketches
            .unwrap_or_default()
            .is_empty()
    );
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: 1,
        })
        .unwrap();
    assert!(client.sketch().get(&SketchArgs::default()).unwrap().editing);
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: 2,
        })
        .unwrap();
    assert!(!client.sketch().get(&SketchArgs::default()).unwrap().editing);
    drop(client);
    owner.poll(32).unwrap();
    assert!(owner.active_sketch_edit().is_none());
}

#[test]
fn vetoed_undo_does_not_restore_an_editor_or_move_the_history_cursor() {
    let (mut owner, _) =
        super::tool_veto::veto_owner_for("Finish Sketch", EVENT_TRANSACTION_UNDONE);
    owner.create_part_document().unwrap();
    let document = owner.files.workspace().active_document().unwrap();
    owner
        .create_sketch_for_edit(document, &super::sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    assert!(
        Client::new(owner.document_host())
            .transactions()
            .undo()
            .unwrap_err()
            .to_string()
            .contains("unlocked")
    );
    owner.poll(32).unwrap();
    assert!(owner.active_sketch_edit().is_none());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .history_entries(),
        before
    );
}
