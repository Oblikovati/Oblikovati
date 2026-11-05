// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{client::Caller, wire::DeleteConstraintArgs};
fn delete_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    index: i64,
) -> Result<OKResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .constrain()
        .delete(&DeleteConstraintArgs {
            sketch_index: 0,
            constraint_index: index,
        })
}
#[test]
fn deletion_resolves_geometric_index_once_and_preserves_dimensions_and_survivor_ids() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_filtered_deletion(kind);
    }
}
fn assert_filtered_deletion(kind: DocumentType) {
    let (mut files, document, mut source) = installed(kind);
    let before = owned_geometry(&files, document).sketches()[0]
        .geometry()
        .constraints()
        .to_vec();
    let expected = before
        .iter()
        .filter(|row| row.id != before[1].id)
        .cloned()
        .collect::<Vec<_>>();
    let focus = files.workspace().active_document();
    assert!(delete_call(&mut files, &mut source, 0).unwrap().ok);
    let after = owned_geometry(&files, document).sketches()[0]
        .geometry()
        .constraints();
    assert_eq!(after, expected);
    assert_deleted_owner(&files, document, focus);
}
#[test]
fn malformed_and_out_of_range_deletions_leave_the_owner_unchanged() {
    let (mut files, document, mut source) = installed(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let expected = list(&files);
    for index in [-1, 18, i64::MAX] {
        expect_api_decline(
            delete_call(&mut files, &mut source, index),
            &index.to_string(),
        );
    }
    let host = FileHost::new(&mut files, &mut source, "test");
    for input in [b"{".as_slice(), b"[]", br#"{"constraintIndex":"bad"}"#] {
        expect_api_decline(
            host.call(METHOD_SKETCH_DELETE_CONSTRAINT, input),
            METHOD_SKETCH_DELETE_CONSTRAINT,
        );
    }
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(list(&files), expected);
}
#[test]
fn deletion_undo_redo_and_reopen_keep_ordered_rows_and_dimensions() {
    let (mut files, document, mut source) = installed(DOCUMENT_PART);
    let before = list(&files);
    delete_call(&mut files, &mut source, 5).unwrap();
    let deleted = list(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(list(&files), before);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(list(&files), deleted);
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(list(&files), deleted);
    assert_eq!(
        owned_geometry(&files, reopened).sketches()[0]
            .geometry()
            .constraints()
            .len(),
        20
    );
}
#[test]
fn commit_veto_restores_the_relation_and_emits_an_abort_generation() {
    let (mut files, document, mut source) = installed(DOCUMENT_PART);
    let document = reopen_with_observer(
        &mut files,
        document,
        &mut source,
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    let before = owner_revision(&files, document);
    let expected = list(&files);
    expect_api_decline(delete_call(&mut files, &mut source, 0), "locked");
    assert_eq!(list(&files), expected);
    let after = owner_revision(&files, document);
    // The existing transaction contract advances generation on commit and abort notifications.
    assert_eq!(after.0, before.0 + 2);
    assert_eq!((after.1, after.2), (before.1, before.2));
}
#[test]
fn structural_origin_sources_are_not_deletable_rows_and_wrong_owners_are_refused() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .create(&Default::default())
        .unwrap();
    let before = owner_revision(&files, document);
    expect_api_decline(delete_call(&mut files, &mut source, 0), "0..0");
    assert_eq!(owner_revision(&files, document), before);
    let (mut drawing, _, mut source) = configured_files(DOCUMENT_DRAWING, Default::default());
    expect_api_decline(
        delete_call(&mut drawing, &mut source, 0),
        "part or assembly",
    );
    assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_DELETE_CONSTRAINT));
}

fn assert_deleted_owner(files: &DiscoveryWorkspace, document: u64, focus: Option<u64>) {
    assert_eq!(list(files)[0].kind, "horizontalAlign");
    assert_eq!(list(files).len(), 17);
    assert_eq!(files.workspace().active_document(), focus);
    assert_eq!(
        owner_revision(files, document).1.last().unwrap(),
        "Delete sketch constraint"
    );
}

#[test]
fn all_geometric_rows_can_be_deleted_in_reverse_order_without_removing_dimensions() {
    let (mut files, document, mut source) = installed(DOCUMENT_PART);
    let dimensions = dimension_rows(&files, document);
    for index in (0..18).rev() {
        assert!(delete_call(&mut files, &mut source, index).unwrap().ok);
        assert_eq!(list(&files).len(), index as usize);
    }
    assert_eq!(
        owned_geometry(&files, document).sketches()[0]
            .geometry()
            .constraints(),
        dimensions
    );
}
#[test]
fn stable_workspace_command_deletes_the_selected_id_with_the_same_history_boundary() {
    let (mut files, document, _) = installed(DOCUMENT_PART);
    let geometry = owned_geometry(&files, document).sketches()[0].geometry();
    let selected = geometry.geometric_constraints().nth(5).unwrap().id;
    let removed = files
        .commands(document)
        .unwrap()
        .delete_sketch_geometric_constraint(DocumentSketchId(1), selected)
        .unwrap();
    assert_eq!(removed.id, selected);
    assert_eq!(list(&files).len(), 17);
    let error = files
        .commands(document)
        .unwrap()
        .delete_sketch_geometric_constraint(DocumentSketchId(1), selected)
        .unwrap_err();
    assert!(error.to_string().contains(&selected.0.to_string()));
}

fn dimension_rows(
    files: &DiscoveryWorkspace,
    document: u64,
) -> Vec<oblikovati_model::sketch::SketchConstraint> {
    owned_geometry(files, document).sketches()[0]
        .geometry()
        .constraints()
        .iter()
        .filter(|constraint| matches!(constraint.relation, Radius { .. }))
        .cloned()
        .collect()
}
