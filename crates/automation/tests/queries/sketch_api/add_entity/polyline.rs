// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn polyline_request(closed: bool) -> AddSketchEntityArgs {
    AddSketchEntityArgs {
        closed,
        construction: true,
        centerline: true,
        variant: "ignored-as-in-Go".into(),
        ..literal_input("polyline", &[[0.0; 2], [2.0, 0.0], [2.0, 2.0]])
    }
}
#[test]
fn typed_polyline_creation_returns_ordered_shared_points_and_all_lines() {
    for closed in [false, true] {
        let (mut files, document, mut source) = fresh();
        let reply = add_call(&mut files, &mut source, &polyline_request(closed)).unwrap();
        let expected: Vec<u64> = if closed { vec![5, 6, 7] } else { vec![5, 6] };
        assert_eq!(reply.entity_id, 5);
        assert_eq!(reply.kind, "polyline");
        assert_eq!(reply.entity_i_ds, Some(expected.clone()));
        assert_eq!(reply.point_i_ds, Some(vec![2, 3, 4]));
        assert!(reply.inferred_constraints.is_none());
        verify_polyline_rows(&files, document, closed, &expected);
    }
}
fn verify_polyline_rows(files: &DiscoveryWorkspace, document: u64, closed: bool, ids: &[u64]) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let mut endpoints = vec![[2, 3], [3, 4]];
    if closed {
        endpoints.push([4, 2]);
    }
    assert_eq!(ids.len(), endpoints.len());
    for (id, expected) in ids.iter().zip(endpoints) {
        assert_eq!(
            geometry.line(SketchEntityId(*id)).unwrap().endpoints,
            expected.map(SketchEntityId)
        );
        let info = row(files, *id);
        assert!(info.construction);
        assert!(!geometry.is_centerline(SketchEntityId(*id)).unwrap());
    }
    assert_eq!(rows(files).len(), ids.len() + 1);
    verify_keys(files, &rows(files));
}
#[test]
fn polyline_undo_redo_reopen_and_copy_retain_shared_topology_and_new_namespaces() {
    let (mut files, document, mut source) = fresh();
    let before = rows(&files);
    add_call(&mut files, &mut source, &polyline_request(true)).unwrap();
    let created = rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), before);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(rows(&files), created);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), created);
    activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/polyline-copy.obk"),
    );
    assert_copy_namespaces(&created, &rows(&files));
    verify_keys(&files, &rows(&files));
}
#[test]
fn polyline_rejects_bad_counts_and_coordinates_before_any_identity_reservation() {
    let (mut files, document, mut source) = fresh();
    for (closed, points) in [(false, vec![[0.0; 2]]), (true, vec![[0.0; 2]; 2])] {
        let input = AddSketchEntityArgs {
            closed,
            ..literal_input("polyline", &points)
        };
        assert_refused(&mut files, document, &mut source, &input, "at least");
    }
    let mut input = polyline_request(true);
    input.points.as_mut().unwrap()[2] = None;
    assert_refused(&mut files, document, &mut source, &input, "expected");
    assert_next_point(&mut files, &mut source, 2);
}
#[test]
fn polyline_veto_and_abort_discard_whole_chains_and_persist_retired_ids() {
    let (mut files, document, mut source) = fresh();
    let before = rows(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Abort chain")
        .unwrap();
    add_call(&mut files, &mut source, &polyline_request(true)).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(rows(&files), before);
    let document = reopen_with_observer(
        &mut files,
        document,
        &mut source,
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    expect_api_decline(
        add_call(&mut files, &mut source, &polyline_request(true)),
        "locked",
    );
    assert_eq!(rows(&files), before);
    assert!(files.workspace().needs_save(document).unwrap());
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_next_point(&mut files, &mut source, 14);
}
#[test]
fn polyline_expression_seeds_override_literals_and_use_document_working_units() {
    let (mut files, document, mut source) = configured_scaled(0.001);
    create_empty_sketch(&mut files, &mut source);
    let mut input = polyline_request(false);
    input.point_exprs = Some(vec![
        Some(vec!["0 mm".into(), "0 mm".into()]),
        Some(vec!["20 mm".into(), "0 mm".into()]),
    ]);
    let created = add_call(&mut files, &mut source, &input).unwrap();
    assert_eq!(
        row(&files, created.entity_id).points,
        expected_points(&[[0.0; 2], [2.0, 0.0]])
    );
    let geometry = owned_geometry(&files, document).sketches()[0].geometry();
    let last = SketchEntityId(*created.point_i_ds.as_ref().unwrap().last().unwrap());
    assert_eq!(geometry.point(last).unwrap().position, [2000.0, 0.0]);
}
#[test]
fn polyline_supports_both_sketch_owners_and_read_only_calls_decline() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, _, mut source) = configured_files(kind, Default::default());
        create_empty_sketch(&mut files, &mut source);
        let request = polyline_request(false);
        assert_eq!(
            add_call(&mut files, &mut source, &request)
                .unwrap()
                .entity_i_ds
                .unwrap()
                .len(),
            2
        );
        expect_api_decline(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .add_entity(&request),
            "sketch.addEntity",
        );
    }
}
#[test]
fn workspace_polyline_command_returns_the_same_owned_creation_report() {
    let (mut files, document, _) = fresh();
    let sketch = owned_geometry(&files, document).sketches()[0].id();
    let input = oblikovati_model::history::SketchPolylineCreation {
        points: vec![[0.0; 2], [1.0, 0.0]],
        closed: false,
        construction: false,
    };
    let report = files
        .commands(document)
        .unwrap()
        .create_sketch_polyline(sketch, &input)
        .unwrap();
    assert_eq!(
        report.points.iter().map(|id| id.0).collect::<Vec<_>>(),
        [2, 3]
    );
    assert_eq!(report.lines.iter().map(|id| id.0).collect::<Vec<_>>(), [4]);
    assert!(!row(&files, 4).construction);
}
