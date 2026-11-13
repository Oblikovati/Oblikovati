// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::drawing::{
    DrawingAnnotationDefinition, DrawingBaseViewDefinition, DrawingDocument, DrawingSheetRecipe,
};
type DrawingHost = FileWorkspace<ContentObserver, RecordingWorkspace, WorkspaceFiles>;
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
            )?;
            drawing.add_annotation_definition(
                1,
                DrawingAnnotationDefinition::parts_list("Parts", [250., 180.])?,
            )
        })
        .unwrap();
}
#[test]
fn deferred_open_rejects_missing_geometry_before_changing_active_document() {
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(1000);
    let drawing = host
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_DRAWING, "Drawing1"),
            ContentObserver,
            "alpha",
            &mut source,
        )
        .unwrap()
        .session_id;
    let target = path("no-saved-views.odd");
    host.save_as(drawing, &target, "alpha", &mut source)
        .unwrap();
    host.close_discarding_changes(drawing).unwrap();
    let part = host
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_PART, "Part1"),
            ContentObserver,
            "alpha",
            &mut source,
        )
        .unwrap()
        .session_id;
    let result =
        host.open_deferred_with_factory(&target, || Ok(ContentObserver), "alpha", &mut source);
    assert!(
        result
            .unwrap_err()
            .to_string()
            .contains("expected saved drawing view geometry")
    );
    assert_eq!(host.workspace().active_document(), Some(part));
    assert_eq!(host.workspace().documents(), &[part]);
}
#[test]
fn drawing_content_is_owned_by_history_and_survives_save_close_and_reopen() {
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(1000);
    let created = host
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_DRAWING, "Drawing1"),
            ContentObserver,
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
            ContentObserver,
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
    assert_eq!(
        drawing_state(host, reopened).sheets()[1].annotations[0].name(),
        "Parts"
    );
    assert!(!host.workspace().needs_save(reopened).unwrap());
}

#[test]
fn staged_view_geometry_is_saved_atomically_and_rejects_a_stale_recipe() {
    use oblikovati_persistence::package::{SavedDrawingLine, SavedDrawingView, SavedDrawingViews};
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), WorkspaceFiles::default());
    let mut source = SequenceGuids(1000);
    let drawing = host
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_DRAWING, "Drawing1"),
            ContentObserver,
            "alpha",
            &mut source,
        )
        .unwrap()
        .session_id;
    assert_eq!(host.saved_drawing_views(drawing).unwrap(), None);
    assert!(host.defer_drawing_updates(drawing).is_err());
    author_view(&mut host, drawing);
    let views = SavedDrawingViews {
        version: SavedDrawingViews::VERSION,
        drawing_recipe: drawing_state(&host, drawing).recipe().clone(),
        views: vec![SavedDrawingView {
            sheet: "Second".into(),
            view: "Front".into(),
            lines: vec![SavedDrawingLine {
                points: vec![[1., 2.], [3., 4.]],
                hidden: false,
            }],
        }],
    };
    assert!(host.keep_drawing_views(u64::MAX, views.clone()).is_err());
    host.keep_drawing_views(drawing, views.clone()).unwrap();
    let target = path("cached-drawing.odd");
    host.save_as(drawing, &target, "alpha", &mut source)
        .unwrap();
    let bytes = host.files().contents.borrow()[&target].clone();
    let package = DocumentPackage::decode(&bytes).unwrap();
    assert_eq!(
        package.restore_saved_drawing_views().unwrap(),
        Some(views.clone())
    );
    assert_eq!(
        host.saved_drawing_views(drawing).unwrap(),
        Some(views.clone())
    );
    assert_eq!(host.deferred_drawing_views(drawing).unwrap(), None);
    assert!(host.saved_drawing_views(u64::MAX).is_err());
    let copy = path("cached-copy.odd");
    host.save_copy(drawing, &copy, None, "alpha", &mut source)
        .unwrap();
    let copy_bytes = host.files().contents.borrow()[&copy].clone();
    assert_eq!(
        DocumentPackage::decode(&copy_bytes)
            .unwrap()
            .restore_saved_drawing_views()
            .unwrap(),
        Some(views.clone())
    );
    host.commands(drawing)
        .unwrap()
        .run("Change reference", |snapshot, _| {
            snapshot
                .drawing
                .as_mut()
                .unwrap()
                .set_model_reference("/project/other.opd")
        })
        .unwrap();
    assert_eq!(
        host.saved_drawing_views(drawing).unwrap(),
        Some(views.clone())
    );
    assert!(host.defer_drawing_updates(drawing).is_err());
    assert!(host.save(drawing, "alpha", &mut source).is_err());
    assert_eq!(host.files().contents.borrow()[&target], bytes);
    let mut updated = views;
    updated.drawing_recipe = drawing_state(&host, drawing).recipe().clone();
    host.keep_drawing_views(drawing, updated.clone()).unwrap();
    host.close_saving_changes(drawing, "alpha", &mut source)
        .unwrap();
    let saved = host.files().contents.borrow()[&target].clone();
    assert_eq!(
        DocumentPackage::decode(&saved)
            .unwrap()
            .restore_saved_drawing_views()
            .unwrap(),
        Some(updated.clone())
    );
    let reopened = host
        .open_deferred_with_factory(&target, || Ok(ContentObserver), "alpha", &mut source)
        .unwrap()
        .session_id();
    assert_eq!(
        host.saved_drawing_views(reopened).unwrap(),
        Some(updated.clone())
    );
    assert_eq!(
        host.deferred_drawing_views(reopened).unwrap(),
        Some(&updated)
    );
    host.commands(reopened)
        .unwrap()
        .run("Change drawing", |snapshot, _| {
            snapshot
                .drawing
                .as_mut()
                .unwrap()
                .set_model_reference("/project/new.opd")
        })
        .unwrap();
    assert_eq!(
        host.deferred_drawing_views(reopened).unwrap(),
        Some(&updated)
    );
    host.close_discarding_changes(reopened).unwrap();
    assert!(host.deferred_drawing_views(reopened).is_err());
}
