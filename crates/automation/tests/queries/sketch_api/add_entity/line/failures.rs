// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn wrong_line_counts_and_null_coordinates_decline_before_reserving() {
    let (mut files, document, mut source) = fresh();
    for points in bad_point_shapes() {
        let mut input = line_input([[0.0; 2], [1.0, 1.0]]);
        input.points = points;
        assert_refused(&mut files, document, &mut source, &input, "expected");
    }
    let request = literal_input("line", &[[0.0; 2]; 3]);
    assert_refused(&mut files, document, &mut source, &request, "exactly 2");
    assert_next_point(&mut files, &mut source, 2);
}
#[test]
fn invalid_line_expressions_and_indices_leave_geometry_and_reservations_unchanged() {
    let (mut files, document, mut source) = fresh();
    let mut input = line_input([[0.0; 2], [1.0, 1.0]]);
    input.point_exprs = Some(invalid_line_expressions());
    assert_refused(
        &mut files,
        document,
        &mut source,
        &input,
        "healthy finite length",
    );
    input.point_exprs = None;
    for index in [-1, 1] {
        input.sketch_index = index;
        assert_refused(&mut files, document, &mut source, &input, "ordered index");
    }
    assert_next_point(&mut files, &mut source, 2);
}
#[test]
fn vetoed_line_creation_cannot_publish_geometry_and_retired_ids_survive_reopen() {
    let (mut files, document, mut source) = fresh();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let original = rows(&files);
    refuse_locked_line(&mut files, &mut source);
    assert_eq!(rows(&files), original);
    assert!(files.workspace().needs_save(document).unwrap());
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    let reply = add_line(&mut files, &mut source, [[0.0; 2], [4.0, 0.01]]);
    assert_eq!(reply.point_i_ds.as_deref(), Some([5, 6].as_slice()));
    assert_eq!(reply.entity_id, 7);
}

fn invalid_line_expressions() -> Vec<Option<Vec<String>>> {
    vec![
        Some(vec!["missing".into(), "0 mm".into()]),
        Some(vec!["1 cm".into(), "0 mm".into()]),
    ]
}

fn refuse_locked_line(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let input = line_input([[0.0; 2], [4.0, 0.01]]);
    expect_api_decline(add_call(files, source, &input), "locked");
}
