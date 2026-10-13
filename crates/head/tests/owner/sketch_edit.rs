// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    history::DocumentSketchCreation,
    sketch::SketchPlane,
};
fn request(plane: SketchPlane) -> DocumentSketchCreation {
    DocumentSketchCreation {
        name: "Edited sketch".into(),
        placement: SketchPlacement::Fixed(plane),
        orientation: None,
        project_origin: true,
    }
}
#[test]
fn creation_enters_once_and_finish_preserves_independent_history() {
    let (mut owner, awake) = owner();
    assert!(owner.finish_sketch_edit().is_err());
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &request(SketchPlane::xy()))
        .unwrap();
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    assert!(owner.begin_sketch_edit(document, sketch).is_err());
    assert!(
        owner
            .create_sketch_for_edit(document, &request(SketchPlane::xy()))
            .is_err()
    );
    let history = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    owner.finish_sketch_edit().unwrap();
    assert!(owner.active_sketch_edit().is_none());
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .history_entries(),
        history
    );
    owner.begin_sketch_edit(document, sketch).unwrap();
    owner.start_slot_placement(document, sketch, 2.).unwrap();
    owner.finish_sketch_edit().unwrap();
    assert!(!owner.placement_active());
}
#[test]
fn history_reattaches_identity_and_undoing_creation_exits() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &request(SketchPlane::xy()))
        .unwrap();
    super::commands::dirty_property(&mut owner, document);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    owner.poll(32).unwrap();
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(owner.active_sketch_edit().is_none());
    owner.poll(32).unwrap();
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    owner.poll(32).unwrap();
    assert!(owner.active_sketch_edit().is_none());
    owner.begin_sketch_edit(document, sketch).unwrap();
}
#[test]
fn invalid_entry_and_document_switch_do_not_select_an_arbitrary_sketch() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    assert!(
        owner
            .begin_sketch_edit(first, DocumentSketchId(999))
            .is_err()
    );
    assert!(owner.active_sketch_edit().is_none());
    let sketch = owner
        .create_sketch_for_edit(first, &request(SketchPlane::xy()))
        .unwrap();
    let second = create(&mut owner, &awake);
    assert!(owner.active_sketch_edit().is_none());
    assert!(owner.begin_sketch_edit(first, sketch).is_err());
    assert!(owner.begin_sketch_edit(second, sketch).is_err());
    assert!(owner.placement_preview().is_none());
}
#[test]
fn creation_failure_and_transaction_refusal_preserve_edit_state() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let mut invalid = request(SketchPlane::xy());
    invalid.placement = SketchPlacement::WorkPlane(oblikovati_model::feature::FeatureId(999));
    assert!(owner.create_sketch_for_edit(document, &invalid).is_err());
    assert!(owner.active_sketch_edit().is_none());
    let sketch = owner
        .create_sketch_for_edit(document, &request(SketchPlane::xy()))
        .unwrap();
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Open".into(),
        })
        .unwrap();
    assert!(owner.finish_sketch_edit().is_err());
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
    owner.finish_sketch_edit().unwrap();
}

#[test]
fn scripts_forward_selected_sketch_and_view_refusal() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &request(SketchPlane::xy()))
        .unwrap();
    let engine =
        oblikovati_script::IsolatedLuaEngine::new(&std::env::current_exe().unwrap(), Vec::new());
    let mut scripted =
        crate::DesktopScripts::new(owner, engine, oblikovati_script::ScriptLimits::gui());
    assert_eq!(scripted.sketch_edit().unwrap().sketch, sketch);
    scripted
        .decline_sketch_edit("named test refusal".into())
        .unwrap();
    assert!(scripted.sketch_edit().is_none());
    assert!(
        scripted
            .take_command_notice()
            .unwrap()
            .message()
            .contains("named test refusal")
    );
    scripted.finish().unwrap();
}

#[test]
fn only_part_and_assembly_host_edits_and_assembly_does_not_project_part_origin() {
    for kind in ["part", "assembly", "drawing", "presentation"] {
        let (mut owner, awake) = owner();
        let document = invoke(&mut owner, &awake, move |client| {
            client
                .documents()
                .create(&CreateDocumentArgs {
                    type_: kind.into(),
                    name: "Host".into(),
                    ..Default::default()
                })
                .unwrap()
                .id
        });
        let result = owner.create_sketch_for_edit(document, &request(SketchPlane::xy()));
        if matches!(kind, "drawing" | "presentation") {
            assert!(result.unwrap_err().contains("expected a part or assembly"));
            assert!(owner.active_sketch_edit().is_none());
            continue;
        }
        let sketch = result.unwrap();
        let snapshot = owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot();
        let count = snapshot
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .points()
            .len();
        assert_eq!(count, usize::from(kind == "part"));
    }
}

#[test]
fn undo_redo_in_one_host_borrow_does_not_revive_a_removed_edit() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    owner
        .create_sketch_for_edit(document, &request(SketchPlane::xy()))
        .unwrap();
    {
        let client = Client::new(owner.document_host());
        client.transactions().undo().unwrap();
        client.transactions().redo().unwrap();
    }
    assert!(
        owner.active_sketch_edit().is_none(),
        "redo must not reopen an edit ended by undoing its creation"
    );
}

#[test]
fn switching_away_and_back_in_one_host_borrow_does_not_revive_edit() {
    let (mut owner, awake) = owner();
    let second = create(&mut owner, &awake);
    let first = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(first, &request(SketchPlane::xy()))
        .unwrap();
    owner.start_slot_placement(first, sketch, 2.).unwrap();
    {
        let client = Client::new(owner.document_host());
        client
            .documents()
            .activate(&oblikovati_api::wire::ActivateDocumentArgs { id: first })
            .unwrap();
        assert!(
            client
                .documents()
                .activate(&oblikovati_api::wire::ActivateDocumentArgs { id: 999 })
                .is_err()
        );
    }
    assert!(owner.active_sketch_edit().is_some());
    {
        let client = Client::new(owner.document_host());
        client
            .documents()
            .activate(&oblikovati_api::wire::ActivateDocumentArgs { id: second })
            .unwrap();
        client
            .documents()
            .activate(&oblikovati_api::wire::ActivateDocumentArgs { id: first })
            .unwrap();
    }
    assert!(owner.active_sketch_edit().is_none());
    assert!(!owner.placement_active());
}
