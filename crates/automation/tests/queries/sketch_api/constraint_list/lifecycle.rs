// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::client::Caller;
#[test]
fn failed_dimension_health_is_inspectable_without_solve_or_any_owner_mutation() {
    let (mut files, document, mut source) = installed(DOCUMENT_PART);
    assert_failed_evaluation(&files, document);
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    let focus = files.workspace().active_document();
    assert_eq!(list(&files).len(), 18);
    assert_eq!(file_host_rows(&mut files, &mut source), list(&files));
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(files.workspace().active_document(), focus);
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
}
#[test]
fn empty_geometric_enumeration_is_an_array_and_excludes_generated_origin_sources() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .create(&Default::default())
        .unwrap();
    let bytes = FileQueryHost::new(&files)
        .call(METHOD_SKETCH_CONSTRAINTS, b"{}")
        .unwrap();
    let expected = serde_json::json!({"constraints": []});
    assert_eq!(
        serde_json::from_slice::<serde_json::Value>(&bytes).unwrap(),
        expected
    );
    assert!(FILE_QUERY_METHODS.contains(&METHOD_SKETCH_CONSTRAINTS));
}
#[test]
fn ordered_index_malformed_input_and_wrong_document_kind_are_named_declines() {
    let (files, document, _) = installed(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let client = Client::new(FileQueryHost::new(&files));
    for sketch_index in [-1, 1, i64::MAX] {
        expect_api_decline(
            client.sketch().constraints(&SketchArgs { sketch_index }),
            "ordered index",
        );
    }
    assert_malformed_queries(&files);
    assert_eq!(owner_revision(&files, document), before);
    let (drawing, _, _) = configured_files(DOCUMENT_DRAWING, Default::default());
    assert_query_decline(&drawing, "part or assembly document");
}
#[test]
fn save_reopen_copy_and_undo_redo_preserve_the_geometric_projection() {
    let (mut files, document, mut source) = installed(DOCUMENT_PART);
    let expected = list(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_query_decline(&files, "ordered index");
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(list(&files), expected);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(list(&files), expected);
    super::super::entities::activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/constraint-copy.obk"),
    );
    assert_eq!(list(&files), expected);
}
#[test]
fn inferred_line_reply_index_addresses_the_same_canonical_geometric_row() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    client.sketch().create(&Default::default()).unwrap();
    let result = client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: "line".into(),
            points: Some(vec![Some(vec![0.0, 0.0]), Some(vec![4.0, 0.01])]),
            ..Default::default()
        })
        .unwrap();
    let inference = &result.inferred_constraints.as_ref().unwrap()[0];
    let rows = list(&files);
    let row = &rows[usize::try_from(inference.constraint_index).unwrap()];
    assert_eq!(row.kind, inference.kind);
    assert_eq!(row.entities, inference.entities);
    assert_eq!(rows[0].kind, "coincident");
}

fn assert_query_decline(files: &DiscoveryWorkspace, expected: &str) {
    let result = Client::new(FileQueryHost::new(files))
        .sketch()
        .constraints(&Default::default());
    expect_api_decline(result, expected);
}

fn assert_malformed_queries(files: &DiscoveryWorkspace) {
    let host = FileQueryHost::new(files);
    for input in [b"{".as_slice(), b"[]", b"{\"sketchIndex\":\"bad\"}"] {
        expect_api_decline(
            host.call(METHOD_SKETCH_CONSTRAINTS, input),
            METHOD_SKETCH_CONSTRAINTS,
        );
    }
}

fn assert_failed_evaluation(files: &DiscoveryWorkspace, document: u64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let health = snapshot.sketches.sketches()[0].health(&snapshot.parameters);
    assert!(matches!(
        health,
        PlacedSketchHealth::Current(SketchEvaluation::Failed(_))
    ));
}
#[test]
fn source_parameter_edits_do_not_change_authored_constraint_rows() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let expected = list(&files);
    assert_eq!(circle_radius(files.workspace(), document, sketch), 1.0);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&numeric("width", "2 cm"))
        .unwrap();
    let radius = circle_radius(files.workspace(), document, sketch);
    let tolerance = 2.0 * SolveOptions::default().relative_tolerance;
    assert!(
        (radius - 2.0).abs() <= tolerance,
        "radius {radius}: expected 2 within solver tolerance {tolerance}"
    );
    assert_eq!(list(&files), expected);
}
#[test]
fn solved_arc_structural_equations_do_not_become_authored_constraint_rows() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    client.sketch().create(&Default::default()).unwrap();
    client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: "arc".into(),
            points: Some(vec![
                Some(vec![0.0, 0.0]),
                Some(vec![2.0, 0.0]),
                Some(vec![0.0, 2.0]),
            ]),
            ccw: true,
            ..Default::default()
        })
        .unwrap();
    assert!(status_call(&files, 0).unwrap().equations > 0);
    assert!(list(&files).is_empty());
}

fn file_host_rows(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> Vec<ConstraintInfo> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .constraints(&Default::default())
        .unwrap()
        .constraints
        .unwrap()
}
