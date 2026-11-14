// SPDX-License-Identifier: GPL-2.0-only
//! Native owner document and basic sketch lifecycle tests.
use super::*;

#[test]
fn gui_new_part_creates_an_active_unsaved_document() {
    let (mut owner, _) = owner();
    assert!(owner.create_part_document().unwrap());
    let workspace = owner.documents().workspace();
    let first = workspace.active_document().unwrap();
    assert_eq!(workspace.documents(), &[first]);
    assert!(!owner.close_allowed().unwrap());
    assert!(owner.create_part_document().unwrap());
    assert_eq!(owner.documents().workspace().documents().len(), 2);
    assert_ne!(owner.documents().workspace().active_document(), Some(first));
}
#[test]
fn gui_sketch_xy_enters_and_finishes_the_owned_edit() {
    use oblikovati_gui::GuiAuthoringContext::*;
    let (mut owner, _) = owner();
    assert!(owner.edit_sketch_from_gui(true).is_err());
    gui_commands::assert_authoring_context(&owner, NoDocument);
    owner.create_part_document().unwrap();
    gui_commands::assert_authoring_context(&owner, Part);
    assert!(owner.edit_sketch_from_gui(true).unwrap());
    gui_commands::assert_authoring_context(&owner, Sketch);
    let edit = owner.active_sketch_edit().unwrap();
    assert_eq!(edit.plane, oblikovati_model::sketch::SketchPlane::xy());
    assert!(owner.edit_sketch_from_gui(true).is_err());
    assert!(owner.edit_sketch_from_gui(false).unwrap());
    assert!(owner.active_sketch_edit().is_none());
    gui_commands::assert_authoring_context(&owner, Part);
    assert!(owner.edit_sketch_from_gui(false).is_err());
}
#[test]
fn gui_rectangle_requires_an_edit_and_uses_owned_placement() {
    let (mut owner, _) = owner();
    assert!(
        owner
            .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Rectangle)
            .is_err()
    );
    owner.create_part_document().unwrap();
    owner.edit_sketch_from_gui(true).unwrap();
    assert!(
        owner
            .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Rectangle)
            .unwrap()
    );
    assert!(owner.placement_active());
    owner.edit_sketch_from_gui(false).unwrap();
    assert!(!owner.placement_active());
    assert!(
        owner
            .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Rectangle)
            .is_err()
    );
}
#[test]
fn typed_workers_create_save_reopen_and_close_through_the_document_owner() {
    let (mut owner, awake) = owner();
    assert!(owner.close_allowed().unwrap());
    let id = create(&mut owner, &awake);
    assert!(!owner.close_allowed().unwrap());
    assert_eq!(owner.refused_close(), [id]);
    invoke(&mut owner, &awake, move |client| {
        client
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: id,
                new_full_document_name: "/memory/part.opd".into(),
            })
            .unwrap()
    });
    assert!(owner.close_allowed().unwrap());
    verify_transaction_close_guard(&mut owner, &awake);
    verify_close_and_reopen(&mut owner, &awake, id);
}
fn verify_close_and_reopen(owner: &mut TestOwner, awake: &Receiver<()>, id: u64) {
    invoke(owner, awake, move |client| {
        client
            .documents()
            .close(&CloseDocumentArgs { id, force: false })
            .unwrap()
    });
    assert!(owner.documents().workspace().documents().is_empty());
    verify_reopened_owner(owner, awake);
}
fn verify_reopened_owner(owner: &mut TestOwner, awake: &Receiver<()>) {
    let reopened = invoke(owner, awake, |client| {
        client
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: "/memory/part.opd".into(),
                ..Default::default()
            })
            .unwrap()
    });
    assert!(
        owner
            .documents()
            .workspace()
            .documents()
            .contains(&reopened.id)
    );
    assert!(owner.close_allowed().unwrap());
}
fn verify_transaction_close_guard(owner: &mut TestOwner, awake: &Receiver<()>) {
    invoke(owner, awake, |client| {
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "edit".into(),
            })
            .unwrap()
    });
    assert!(!owner.close_allowed().unwrap());
    invoke(owner, awake, |client| {
        client.transactions().abort().unwrap()
    });
    assert!(owner.close_allowed().unwrap());
}
