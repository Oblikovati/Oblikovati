// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART};
use oblikovati_gui::GuiAuthoringContext::{OtherDocument, Sketch};
use oblikovati_model::sketch::SketchPlane;

#[test]
fn ribbon_snapshot_retains_document_kind_through_sketch_context_changes() {
    let (mut owner, awake) = owner();
    assert!(owner.gui_command_state().unwrap().document_type.is_none());
    let document = create(&mut owner, &awake);
    let kind = owner.gui_command_state().unwrap().document_type;
    assert_eq!(kind, Some(DOCUMENT_PART));
    owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    let editing = owner.gui_command_state().unwrap();
    assert_eq!(editing.document_type, Some(DOCUMENT_PART));
    assert_eq!(editing.authoring, Sketch);
    owner.finish_sketch_edit().unwrap();
    let kind = owner.gui_command_state().unwrap().document_type;
    assert_eq!(kind, Some(DOCUMENT_PART));
}
#[test]
fn ribbon_snapshot_reads_assembly_kind_from_the_document_owner() {
    let (mut owner, awake) = owner();
    let document = invoke(&mut owner, &awake, |client| {
        client
            .documents()
            .create(&CreateDocumentArgs {
                type_: "assembly".into(),
                name: "Assembly ribbon".into(),
                ..Default::default()
            })
            .unwrap()
            .id
    });
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.document, Some(document));
    assert_eq!(state.document_type, Some(DOCUMENT_ASSEMBLY));
    assert_eq!(state.authoring, OtherDocument);
}
