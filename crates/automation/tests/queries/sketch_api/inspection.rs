// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn inspection_lists_current_planes_and_reads_the_same_selected_sketch_without_history() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    assert_eq!(client.sketch().list().unwrap().sketches, Some(vec![]));
    for plane in ["XY", "XZ", "YZ"] {
        client
            .sketch()
            .create(&CreateSketchArgs {
                plane: plane.into(),
                ..Default::default()
            })
            .unwrap();
    }
    let before = owner_revision(&files, document);
    let client = Client::new(FileQueryHost::new(&files));
    let rows = client.sketch().list().unwrap().sketches.unwrap();
    for (index, plane) in ["XY", "XZ", "YZ"].into_iter().enumerate() {
        let row = client
            .sketch()
            .get(&SketchArgs {
                sketch_index: i64::try_from(index).unwrap(),
            })
            .unwrap();
        assert_eq!(rows[index], row);
        assert_eq!(row.plane, plane);
        assert!(row.visible && row.healthy);
        assert_eq!(row.dof, 0);
        assert!(!row.editing && !row.consumed && !row.shared);
    }
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn inspection_matches_edit_identity_in_both_owners() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let before = owner_revision(&files, document);
    let edit = Some(inspection_edit(&mut files, document, sketch));
    let read = Client::new(FileQueryHost::new(&files).with_sketch_edit(edit.clone()))
        .sketch()
        .get(&SketchArgs::default())
        .unwrap();
    assert!(read.editing);
    assert_eq!(read.dof, 0);
    let current =
        Client::new(FileHost::new(&mut files, &mut source, "test").with_sketch_edit(edit.clone()))
            .sketch()
            .get(&SketchArgs::default())
            .unwrap();
    assert_eq!(read, current);
    let other = Client::new(FileQueryHost::new(&files).with_sketch_edit(None))
        .sketch()
        .get(&SketchArgs::default())
        .unwrap();
    assert!(!other.editing);
    assert_eq!(owner_revision(&files, document), before);
}

fn inspection_edit(
    files: &mut DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> oblikovati_automation::SketchInspectionEdit {
    let watch = files
        .commands(document)
        .unwrap()
        .watch_sketch_continuity(sketch)
        .unwrap();
    oblikovati_automation::SketchInspectionEdit::new(
        document,
        watch,
        files.workspace().focus_identity(),
    )
}

#[test]
fn one_host_cannot_revive_editing_after_switching_away_and_back() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    let second = files
        .create(
            Path::new("/fixtures/second.opd"),
            package,
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    files.activate(document).unwrap();
    let edit = inspection_edit(&mut files, document, sketch);
    let client =
        Client::new(FileHost::new(&mut files, &mut source, "test").with_sketch_edit(Some(edit)));
    assert!(client.sketch().get(&SketchArgs::default()).unwrap().editing);
    for id in [second, document] {
        client
            .documents()
            .activate(&ActivateDocumentArgs { id })
            .unwrap();
    }
    assert!(!client.sketch().get(&SketchArgs::default()).unwrap().editing);
}

#[test]
fn one_host_cannot_revive_editing_after_deletion_and_undo() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let edit = inspection_edit(&mut files, document, sketch);
    let client =
        Client::new(FileHost::new(&mut files, &mut source, "test").with_sketch_edit(Some(edit)));
    client.sketch().delete(&SketchArgs::default()).unwrap();
    client.transactions().undo().unwrap();
    assert!(!client.sketch().get(&SketchArgs::default()).unwrap().editing);
}

#[test]
fn inspection_refuses_missing_owner_wrong_document_kind_and_invalid_indices() {
    for kind in [DOCUMENT_DRAWING, DOCUMENT_PART] {
        let (files, _, _) = configured_files(kind, Default::default());
        let client = Client::new(FileQueryHost::new(&files));
        if kind == DOCUMENT_DRAWING {
            expect_api_decline(client.sketch().list(), "part or assembly");
        }
        for index in [-1, 0, i64::MAX] {
            assert!(
                client
                    .sketch()
                    .get(&SketchArgs {
                        sketch_index: index
                    })
                    .is_err()
            );
        }
    }
}

#[test]
fn inspection_refuses_a_workspace_without_an_active_document() {
    let files = DiscoveryWorkspace::new_with_sketch_options(
        QueryObserver::default(),
        DiscoveryFiles::default(),
        Default::default(),
    );
    expect_api_decline(
        Client::new(FileQueryHost::new(&files)).sketch().list(),
        "active document is absent",
    );
}

#[test]
fn inspection_reports_free_coordinates_without_solving_or_claiming_zero_dof() {
    let (mut files, document, sketch, _) = files_with_circle(Default::default());
    let center = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .points()[0]
        .id;
    files
        .commands(document)
        .unwrap()
        .run("Free point", |snapshot, _| {
            snapshot
                .sketches
                .replace_geometry(sketch, point_geometry(center, vec![]))
        })
        .unwrap();
    let before = owner_revision(&files, document);
    let row = Client::new(FileQueryHost::new(&files))
        .sketch()
        .get(&Default::default())
        .unwrap();
    assert_eq!((row.entity_count, row.dof), (1, 2));
    assert!(row.healthy);
    assert_eq!(owner_revision(&files, document), before);
}
