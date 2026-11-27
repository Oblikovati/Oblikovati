// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    json::EmptyObject,
    wire::{AddDrawingNoteArgs, DeleteAnnotationArgs, DrawingAnnotationInfo},
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};

#[test]
fn drawing_notes_are_independent_sheet_content_with_history() {
    let (mut owner, _) = owner();
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let client = Client::new(owner.document_host());
    assert!(listed(&client).is_none());
    let [plain, leader] = add_notes(&client);
    assert_eq!(listed(&client).unwrap(), [plain.clone(), leader.clone()]);
    delete_and_restore_note(&client, &plain, &leader);
    drop(client);
    let reopened = verify_deferred_and_reopen(&mut owner, drawing, &leader.name);
    assert_eq!(
        listed(&Client::new(owner.document_host())).unwrap(),
        [plain, leader]
    );
    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&reopened].error.is_none());
    let drawing = &state.drawing_documents[&reopened];
    assert!(drawing.model_reference().is_empty());
    let note = drawing.sheets()[0].annotations[1]
        .note_presentation()
        .unwrap();
    assert_eq!(note.anchor_mm, [150., 125.]);
    assert_eq!(
        note.leader_segments_mm,
        [
            [[150., 125.], [120., 125.]],
            [[120., 125.], [123., 126.2]],
            [[120., 125.], [123., 123.8]],
        ]
    );
}

fn add_notes(client: &Client<impl Caller>) -> [DrawingAnnotationInfo; 2] {
    assert!(
        client
            .drawing_annotations()
            .add_note(&AddDrawingNoteArgs::default())
            .is_err()
    );
    let plain = client
        .drawing_annotations()
        .add_note(&AddDrawingNoteArgs {
            name: "Finish".into(),
            xmm: 25.,
            ymm: 80.,
            text: "Finish all surfaces".into(),
            ..Default::default()
        })
        .unwrap()
        .annotation;
    let leader = client
        .drawing_annotations()
        .add_note(&AddDrawingNoteArgs {
            name: plain.name.clone(),
            xmm: 150.,
            ymm: 125.,
            text: "Deburr".into(),
            leader_xmm: 120.,
            leader_ymm: 125.,
        })
        .unwrap()
        .annotation;
    assert_eq!(plain.kind, "drawingNote");
    assert_eq!(plain.tag, "Finish all surfaces");
    assert_eq!(plain.curve_count, 0);
    assert_eq!(leader.name, "NOTE:2");
    assert_eq!(leader.tag, "Deburr");
    assert_eq!(leader.curve_count, 3);
    [plain, leader]
}

fn delete_and_restore_note(
    client: &Client<impl Caller>,
    plain: &DrawingAnnotationInfo,
    leader: &DrawingAnnotationInfo,
) {
    let delete = |name: &str| {
        client
            .drawing_annotations()
            .delete(&DeleteAnnotationArgs { name: name.into() })
    };
    assert!(delete("Missing").is_err());
    assert_eq!(
        delete(&leader.name).unwrap().annotations.as_deref(),
        Some(std::slice::from_ref(plain))
    );
    client.transactions().undo().unwrap();
    assert_eq!(listed(client).unwrap(), [plain.clone(), leader.clone()]);
    client.transactions().redo().unwrap();
    assert_eq!(
        listed(client).unwrap().as_slice(),
        std::slice::from_ref(plain)
    );
    client.transactions().undo().unwrap();
}

fn listed(client: &Client<impl Caller>) -> Option<Vec<DrawingAnnotationInfo>> {
    client
        .drawing_annotations()
        .list(&EmptyObject {})
        .unwrap()
        .annotations
}

fn verify_deferred_and_reopen(owner: &mut TestOwner, drawing: u64, note: &str) -> u64 {
    let path = document_fixture_path!("/memory/sheet-notes.odd");
    let client = Client::new(owner.document_host());
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: drawing,
            new_full_document_name: path.into(),
        })
        .unwrap();
    close_drawing(&client, drawing);
    drop(client);
    let deferred = owner
        .document_host()
        .open_deferred_drawing_path(std::path::Path::new(path))
        .unwrap();
    let client = Client::new(owner.document_host());
    assert_eq!(listed(&client).unwrap().len(), 2);
    assert!(
        client
            .drawing_annotations()
            .delete(&DeleteAnnotationArgs { name: note.into() })
            .is_err()
    );
    close_drawing(&client, deferred);
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id
}

fn close_drawing(client: &Client<impl Caller>, id: u64) {
    client
        .documents()
        .close(&CloseDocumentArgs { id, force: true })
        .unwrap();
}
