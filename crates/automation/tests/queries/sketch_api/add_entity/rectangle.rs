// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn rectangle_request(variant: &str) -> AddSketchEntityArgs {
    let points = if variant == "threePoint" {
        vec![[0.0; 2], [3.0, 0.0], [5.0, 2.0]]
    } else {
        vec![[0.0; 2], [3.0, 2.0]]
    };
    AddSketchEntityArgs {
        variant: variant.into(),
        construction: true,
        centerline: true,
        ..literal_input("rectangle", &points)
    }
}
#[test]
fn rectangle_variants_return_closed_owned_chains_in_go_vertex_order() {
    for variant in ["", "corners", "center", "threePoint", "Go-default"] {
        let (mut files, document, mut source) = fresh();
        let reply = add_call(&mut files, &mut source, &rectangle_request(variant)).unwrap();
        assert_eq!((reply.kind.as_str(), reply.entity_id), ("rectangle", 6));
        assert_eq!(reply.entity_i_ds, Some(vec![6, 7, 8, 9]));
        assert_eq!(reply.point_i_ds, Some(vec![2, 3, 4, 5]));
        assert!(reply.inferred_constraints.is_none());
        verify_rectangle_chain(&files, document, variant);
    }
}
fn verify_rectangle_chain(files: &DiscoveryWorkspace, document: u64, variant: &str) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let points = if variant == "center" {
        [[-3.0, -2.0], [3.0, -2.0], [3.0, 2.0], [-3.0, 2.0]]
    } else {
        [[0.0; 2], [3.0, 0.0], [3.0, 2.0], [0.0, 2.0]]
    };
    for index in 0..4 {
        let id = SketchEntityId(6 + index as u64);
        let line = geometry.line(id).unwrap();
        assert_eq!(
            line.endpoints,
            [
                SketchEntityId(2 + index as u64),
                SketchEntityId(2 + (index as u64 + 1) % 4)
            ]
        );
        assert_eq!(
            row(files, id.0).points,
            expected_points(&[points[index], points[(index + 1) % 4]])
        );
        assert!(geometry.is_construction(id).unwrap());
        assert!(!geometry.is_centerline(id).unwrap());
    }
    assert!(geometry.constraints().is_empty());
    assert_eq!(rows(files).len(), 5);
    verify_keys(files, &rows(files));
}
#[test]
fn invalid_rectangle_seeds_fail_before_identity_reservation() {
    let (mut files, document, mut source) = fresh();
    for (variant, points) in [
        ("center", vec![[0.0; 2]]),
        ("threePoint", vec![[0.0; 2]; 2]),
        ("corners", vec![[0.0; 2]; 3]),
        ("threePoint", vec![[0.0; 2]; 3]),
    ] {
        let input = AddSketchEntityArgs {
            variant: variant.into(),
            ..literal_input("rectangle", &points)
        };
        assert_refused(&mut files, document, &mut source, &input, "expected");
    }
    assert_next_point(&mut files, &mut source, 2);
}
#[test]
fn rectangle_undo_redo_and_reopen_preserve_owned_vertices_and_keys() {
    let (mut files, document, mut source) = fresh();
    let before = rows(&files);
    add_call(&mut files, &mut source, &rectangle_request("threePoint")).unwrap();
    let created = rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), before);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(rows(&files), created);
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), created);
    verify_keys(&files, &rows(&files));
}
#[test]
fn rectangle_abort_and_veto_retire_all_reserved_vertices_and_edges() {
    let (mut files, document, mut source) = fresh();
    let before = rows(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Abort rectangle")
        .unwrap();
    add_call(&mut files, &mut source, &rectangle_request("center")).unwrap();
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
        add_call(&mut files, &mut source, &rectangle_request("center")),
        "locked",
    );
    assert_eq!(rows(&files), before);
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_next_point(&mut files, &mut source, 18);
}
#[test]
fn rectangle_expressions_override_literals_and_apply_working_units() {
    let (mut files, document, mut source) = configured_scaled(0.001);
    create_empty_sketch(&mut files, &mut source);
    let mut input = rectangle_request("corners");
    input.point_exprs = Some(vec![
        Some(vec!["10 mm".into(), "20 mm".into()]),
        Some(vec!["30 mm".into(), "40 mm".into()]),
    ]);
    let report = add_call(&mut files, &mut source, &input).unwrap();
    assert_eq!(
        row(&files, report.entity_id).points,
        expected_points(&[[1.0, 2.0], [3.0, 2.0]])
    );
    let geometry = owned_geometry(&files, document).sketches()[0].geometry();
    assert_eq!(
        geometry
            .point(SketchEntityId(report.point_i_ds.unwrap()[2]))
            .unwrap()
            .position,
        [3000.0, 4000.0]
    );
}
