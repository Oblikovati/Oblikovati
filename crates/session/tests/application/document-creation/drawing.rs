// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::drawing::{DrawingBaseViewDefinition, DrawingDocument, DrawingSheetRecipe};
#[derive(Default)]
struct DrawingObserver;
impl TransactionObserver for DrawingObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}
type DrawingHost = FileWorkspace<DrawingObserver, RecordingWorkspace, WorkspaceFiles>;
fn drawing_state(host: &DrawingHost, id: u64) -> &DrawingDocument {
    host.workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .drawing
        .as_ref()
        .unwrap()
}
fn author_view(host: &mut DrawingHost, id: u64) {
    host.commands(id)
        .unwrap()
        .run("Add referenced drawing view", |snapshot, _| {
            let drawing = snapshot.drawing.as_mut().unwrap();
            drawing.set_model_reference("/project/bracket.opd")?;
            drawing.add_sheet(DrawingSheetRecipe::a3_landscape("Second"))?;
            drawing.add_base_view_definition(
                1,
                DrawingBaseViewDefinition::new("Front", BASE_VIEW_FRONT, 0.5, [90., 120.])?,
            )
        })
        .unwrap();
}
#[test]
fn drawing_content_is_owned_by_history_and_survives_save_close_and_reopen() {
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(1000);
    let created = host
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_DRAWING, "Drawing1"),
            DrawingObserver,
            "alpha",
            &mut source,
        )
        .unwrap()
        .session_id;
    let initial = drawing_state(&host, created).clone();
    host.save_as(created, &path("drawing.odd"), "alpha", &mut source)
        .unwrap();
    assert!(!host.workspace().needs_save(created).unwrap());
    author_view(&mut host, created);
    let expected = drawing_state(&host, created).clone();
    assert!(host.workspace().needs_save(created).unwrap());
    host.commands(created).unwrap().undo().unwrap();
    assert_eq!(drawing_state(&host, created), &initial);
    host.commands(created).unwrap().redo().unwrap();
    assert_eq!(drawing_state(&host, created), &expected);
    verify_reopened_drawing(&mut host, created, &expected, &mut source);
}
fn verify_reopened_drawing(
    host: &mut DrawingHost,
    id: u64,
    expected: &DrawingDocument,
    source: &mut SequenceGuids,
) {
    host.save(id, "alpha", source).unwrap();
    host.close_discarding_changes(id).unwrap();
    let reopened = host
        .open(
            &path("drawing.odd"),
            DrawingObserver,
            "alpha",
            source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    assert_eq!(drawing_state(host, reopened), expected);
    assert_eq!(drawing_state(host, reopened).active_sheet(), 1);
    assert_eq!(
        drawing_state(host, reopened).sheets()[1].views[0].name,
        "Front"
    );
    assert!(!host.workspace().needs_save(reopened).unwrap());
}
