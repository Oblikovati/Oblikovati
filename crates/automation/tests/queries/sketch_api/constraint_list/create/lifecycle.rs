// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::client::Caller;
#[test]
fn creation_save_reopen_copy_and_delete_keep_constraint_marks_and_actual_rows() {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    let first = add(&mut files, &mut source, &request("horizontal", &[7])).unwrap();
    assert_eq!(first.index, 0);
    files.commands(document).unwrap().undo().unwrap();
    assert!(list(&files).is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(list(&files)[0].kind, "horizontal");
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    let second = add(&mut files, &mut source, &request("horizontal", &[8])).unwrap();
    assert_eq!(second.index, 1);
    assert_copy_and_retired_id(&mut files, document, &mut source);
}
fn assert_copy_and_retired_id(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
) {
    let expected = list(files);
    super::super::super::entities::activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/created-constraint-copy.obk"),
    );
    assert_eq!(list(files), expected);
    delete_first(files, source);
    let created = add(files, source, &request("fix", &[2])).unwrap();
    assert_eq!(created.index, 1);
    let document = files.workspace().active_document().unwrap();
    let rows = owned_geometry(files, document).sketches()[0]
        .geometry()
        .constraints();
    assert_eq!(rows[0].id, SketchConstraintId(2));
    assert_eq!(rows[1].id, SketchConstraintId(3));
}
#[test]
fn malformed_payload_and_wrong_sketch_index_do_not_publish_or_allocate() {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    for sketch_index in [-1, 1, i64::MAX] {
        let mut input = request("horizontal", &[7]);
        input.sketch_index = sketch_index;
        expect_api_decline(add(&mut files, &mut source, &input), "ordered index");
    }
    assert_malformed_creation(&mut files, &mut source);
    assert_eq!(owner_revision(&files, document), before);
}
#[test]
fn source_changes_and_mixed_dimensions_return_the_current_rank_and_filtered_index() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&numeric("width", "2 cm"))
        .unwrap();
    let geometry = owned_geometry(&files, document)
        .get(sketch)
        .unwrap()
        .geometry();
    let center = geometry.circles()[0].center.0;
    let result = add(&mut files, &mut source, &request("fix", &[center])).unwrap();
    assert_eq!(result.index, 1);
    assert_eq!(result.dof, 0);
    assert_eq!(result.kind, "fix");
    assert_eq!(list(&files).len(), 2);
    let tolerance = 2.0 * SolveOptions::default().relative_tolerance;
    assert!((circle_radius(files.workspace(), document, sketch) - 2.0).abs() <= tolerance);
}
#[test]
fn late_contact_decline_is_named_and_keeps_the_attempted_id_retired() {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    let before = owned_geometry(&files, document).clone();
    expect_api_decline(
        add(&mut files, &mut source, &request("tangent", &[7, 9])),
        "nonzero seed side",
    );
    assert!(before.same_evaluated_state(owned_geometry(&files, document)));
    let created = add(&mut files, &mut source, &request("horizontal", &[7])).unwrap();
    assert_eq!(created.index, 0);
    assert_eq!(
        owned_geometry(&files, document).sketches()[0]
            .geometry()
            .constraints()[0]
            .id,
        SketchConstraintId(2)
    );
}

fn delete_first(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .constrain()
        .delete(&DeleteConstraintArgs {
            sketch_index: 0,
            constraint_index: 0,
        })
        .unwrap();
}

fn assert_malformed_creation(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let host = FileHost::new(files, source, "test");
    for input in [
        b"{".as_slice(),
        b"[]",
        br#"{"entities":[-1]}"#,
        br#"{"kind":3}"#,
    ] {
        expect_api_decline(
            host.call(METHOD_SKETCH_ADD_CONSTRAINT, input),
            METHOD_SKETCH_ADD_CONSTRAINT,
        );
    }
}

#[test]
fn stable_workspace_creation_uses_the_same_relation_and_owner_boundary() {
    let (mut files, document, _) = fresh(DOCUMENT_PART);
    let input =
        oblikovati_model::sketch::SketchGeometricConstraintCreation::Fix { point: entity(2) };
    let created = files
        .commands(document)
        .unwrap()
        .create_sketch_geometric_constraint(DocumentSketchId(1), &input)
        .unwrap();
    assert_eq!(created.constraint.id, SketchConstraintId(1));
    assert_eq!(created.geometric_index, 0);
    assert_eq!(list(&files)[0].kind, "fix");
}
#[test]
fn drawing_owner_cannot_author_geometric_sketch_constraints() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_DRAWING, Default::default());
    let before = owner_revision(&files, document);
    expect_api_decline(
        add(&mut files, &mut source, &request("fix", &[1])),
        "part or assembly",
    );
    assert_eq!(owner_revision(&files, document), before);
}
