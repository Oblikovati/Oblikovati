// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn polygon_request(variant: &str, sides: i64) -> AddSketchEntityArgs {
    AddSketchEntityArgs {
        variant: variant.into(),
        sides,
        construction: true,
        ..literal_input("polygon", &[[0.0; 2], [3.0, 0.0]])
    }
}
#[test]
fn polygon_api_returns_lines_vertices_and_center_with_real_regularity_constraints() {
    for variant in ["", "inscribed", "circumscribed", "Go-default"] {
        let (mut files, document, mut source) = fresh();
        let reply = add_call(&mut files, &mut source, &polygon_request(variant, 4)).unwrap();
        assert_eq!((reply.entity_id, reply.kind.as_str()), (6, "polygon"));
        assert_eq!(reply.entity_i_ds, Some(vec![6, 7, 8, 9]));
        assert_eq!(reply.point_i_ds, Some(vec![2, 3, 4, 5, 10]));
        let geometry = owned_geometry(&files, document).sketches()[0].geometry();
        assert_eq!(geometry.constraints().len(), 7);
        assert_eq!(
            geometry.circle(SketchEntityId(11)).unwrap().center,
            SketchEntityId(10)
        );
        assert!(geometry.is_construction(SketchEntityId(11)).unwrap());
        verify_keys(&files, &rows(&files));
        let created = rows(&files);
        files.commands(document).unwrap().undo().unwrap();
        assert_eq!(rows(&files).len(), 1);
        files.commands(document).unwrap().redo().unwrap();
        assert_eq!(rows(&files), created);
        reopen_with_observer(&mut files, document, &mut source, Default::default());
        assert_eq!(rows(&files), created);
    }
}
#[test]
fn invalid_polygon_counts_and_zero_radius_reserve_nothing() {
    let (mut files, document, mut source) = fresh();
    for sides in [-1, 0, 1, 2] {
        assert_refused(
            &mut files,
            document,
            &mut source,
            &polygon_request("", sides),
            "expected",
        );
    }
    let mut input = polygon_request("", 4);
    input.points = Some(vec![Some(vec![0.0; 2]); 2]);
    assert_refused(&mut files, document, &mut source, &input, "radius");
    input.points = Some(vec![Some(vec![0.0; 2])]);
    assert_refused(&mut files, document, &mut source, &input, "exactly 2");
    assert_next_point(&mut files, &mut source, 2);
}
#[test]
fn polygon_abort_and_veto_keep_entity_and_constraint_ids_retired() {
    let (mut files, document, mut source) = fresh();
    let before = rows(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Abort polygon")
        .unwrap();
    add_call(&mut files, &mut source, &polygon_request("", 4)).unwrap();
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
        add_call(&mut files, &mut source, &polygon_request("", 4)),
        "locked",
    );
    assert_eq!(rows(&files), before);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    let reply = add_call(&mut files, &mut source, &polygon_request("", 3)).unwrap();
    assert_eq!(reply.point_i_ds, Some(vec![22, 23, 24, 28]));
    let geometry = owned_geometry(&files, document).sketches()[0].geometry();
    assert_eq!(geometry.constraints()[0].id.0, 15);
}
