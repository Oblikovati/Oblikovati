// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
fn create_corner(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    kind: &str,
    second: &str,
) -> AddSketchEntityResult {
    let first = add_call(
        files,
        source,
        &literal_input("line", &[[-5., 0.], [0., 0.]]),
    )
    .unwrap();
    let next = add_call(files, source, &literal_input("line", &[[0., 0.], [0., 5.]])).unwrap();
    let request = AddSketchEntityArgs {
        kind: kind.into(),
        entity_refs: Some(vec![first.entity_id, next.entity_id]),
        radius: "width".into(),
        distance2: second.into(),
        ..Default::default()
    };
    Client::new(
        FileHost::new(files, source, "test")
            .with_sketch_corner_construction(&MartensiteSketchConstruction, Default::default()),
    )
    .sketch()
    .add_entity(&request)
    .unwrap()
}
#[test]
fn fillet_drives_radius_and_survives_history_and_reopen() {
    let (mut files, document, mut source) = fresh();
    let result = create_corner(&mut files, &mut source, "fillet", "");
    assert!((row(&files, result.entity_id).radius - 1.).abs() < 1e-7);
    set_width(&mut files, &mut source, "2 cm");
    assert!((row(&files, result.entity_id).radius - 2.).abs() < 1e-7);
    files.commands(document).unwrap().undo().unwrap();
    assert!((row(&files, result.entity_id).radius - 1.).abs() < 1e-7);
    files.commands(document).unwrap().redo().unwrap();
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert!((row(&files, result.entity_id).radius - 2.).abs() < 1e-7);
    set_width(&mut files, &mut source, "1.5 cm");
    assert!((row(&files, result.entity_id).radius - 1.5).abs() < 1e-7);
    assert_eq!(owned_geometry(&files, reopened).sketches().len(), 1);
}
#[test]
fn chamfer_binds_both_setbacks_to_the_same_default_source() {
    let (mut files, _, mut source) = fresh();
    let result = create_corner(&mut files, &mut source, "chamfer", "");
    let dimensions = Client::new(FileQueryHost::new(&files))
        .sketch()
        .dimensions(&Default::default())
        .unwrap()
        .dimensions
        .unwrap();
    assert_eq!(dimensions.len(), 2);
    assert!(dimensions.iter().all(|d| d.expression == "width"));
    set_width(&mut files, &mut source, "2 cm");
    let points = row(&files, result.entity_id).points.unwrap();
    let a = points[0].as_ref().unwrap();
    let b = points[1].as_ref().unwrap();
    assert!(((a[0] - b[0]).hypot(a[1] - b[1]) - 8f64.sqrt()).abs() < 1e-6);
}
#[test]
fn chamfer_independent_setbacks_recompute_undo_and_reopen() {
    let (mut files, document, mut source) = fresh();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "height".into(),
            expression: "2 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let result = create_corner(&mut files, &mut source, "chamfer", "height");
    assert_setbacks(&files, document, [1., 2.]);
    set_width(&mut files, &mut source, "1.5 cm");
    assert_setbacks(&files, document, [1.5, 2.]);
    files.commands(document).unwrap().undo().unwrap();
    assert_setbacks(&files, document, [1., 2.]);
    files.commands(document).unwrap().redo().unwrap();
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_setbacks(&files, reopened, [1.5, 2.]);
    assert!(row(&files, result.entity_id).points.is_some());
}
fn assert_setbacks(files: &DiscoveryWorkspace, document: u64, expected: [f64; 2]) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let actual: Vec<_> = geometry
        .dimensional_constraints()
        .map(|(constraint, _)| geometry.measure_dimension(constraint.id).unwrap().value)
        .collect();
    assert_eq!(actual.len(), 2);
    for (a, b) in actual.into_iter().zip(expected) {
        assert!((a - b).abs() < 1e-7, "{a} expected {b}");
    }
}
#[test]
fn invalid_chamfer_parameter_edits_are_rejected_without_changing_geometry_or_history() {
    let (mut files, document, mut source) = fresh();
    let result = create_corner(&mut files, &mut source, "chamfer", "");
    let before = row(&files, result.entity_id);
    let history = owner_revision(&files, document);
    for expression in ["0 cm", "-1 cm"] {
        let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .parameters()
            .set(&oblikovati_api::wire::ParameterSetArgs {
                name: "width".into(),
                expression: expression.into(),
                ..Default::default()
            });
        assert!(error.is_err(), "accepted {expression}");
        assert_eq!(row(&files, result.entity_id), before);
        let after = owner_revision(&files, document);
        assert_eq!((&after.1, after.2), (&history.1, history.2));
    }
}
#[test]
fn formula_edits_are_atomic_and_unrelated_parameter_edits_remain_valid() {
    let (mut files, document, mut source) = fresh();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "base".into(),
            expression: "0.5 cm".into(),
            ..Default::default()
        })
        .unwrap();
    set_width(&mut files, &mut source, "base * 2");
    let result = create_corner(&mut files, &mut source, "chamfer", "");
    let before = row(&files, result.entity_id);
    let history = owner_revision(&files, document);
    let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "base".into(),
            expression: "-1 cm".into(),
            ..Default::default()
        });
    assert!(error.is_err());
    assert_eq!(row(&files, result.entity_id), before);
    let after = owner_revision(&files, document);
    assert_eq!((&after.1, after.2), (&history.1, history.2));
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "unrelated".into(),
            expression: "3 cm".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(row(&files, result.entity_id), before);
}
#[test]
fn direct_session_parameter_edit_cannot_bypass_corner_validation() {
    let (mut files, document, mut source) = fresh();
    let result = create_corner(&mut files, &mut source, "chamfer", "");
    let before = row(&files, result.entity_id);
    let width = owned_parameters(&files, document).find("width").unwrap();
    let error = files
        .commands(document)
        .unwrap()
        .run("Invalid width", |snapshot, _| {
            snapshot.parameters.set_expression(width, "0 cm")
        });
    assert!(error.is_err());
    assert_eq!(row(&files, result.entity_id), before);
}
#[test]
fn oversized_fillet_and_chamfer_edits_leave_fixed_sources_unchanged() {
    for kind in ["fillet", "chamfer"] {
        let (mut files, document, mut source) = fresh();
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .sketch()
            .set_inference_options(&oblikovati_api::wire::InferenceOptionsView {
                infer_enabled: false,
                constrain_enabled: false,
                priority: "none".into(),
            })
            .unwrap();
        let first = add_call(
            &mut files,
            &mut source,
            &literal_input("line", &[[-5., 0.], [0., 0.]]),
        )
        .unwrap();
        let next = add_call(
            &mut files,
            &mut source,
            &literal_input("line", &[[0., 0.], [0., 5.]]),
        )
        .unwrap();
        for point in [
            first.point_i_ds.as_ref().unwrap()[0],
            first.point_i_ds.as_ref().unwrap()[1],
            next.point_i_ds.as_ref().unwrap()[1],
        ] {
            Client::new(FileHost::new(&mut files, &mut source, "test"))
                .constrain()
                .add(&oblikovati_api::wire::AddConstraintArgs {
                    kind: "fix".into(),
                    entities: Some(vec![point]),
                    ..Default::default()
                })
                .unwrap();
        }
        let request = AddSketchEntityArgs {
            kind: kind.into(),
            entity_refs: Some(vec![first.entity_id, next.entity_id]),
            radius: "width".into(),
            ..Default::default()
        };
        let result = Client::new(
            FileHost::new(&mut files, &mut source, "test")
                .with_sketch_corner_construction(&MartensiteSketchConstruction, Default::default()),
        )
        .sketch()
        .add_entity(&request)
        .unwrap();
        let before = row(&files, result.entity_id);
        let history = owner_revision(&files, document);
        let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .parameters()
            .set(&oblikovati_api::wire::ParameterSetArgs {
                name: "width".into(),
                expression: "10 cm".into(),
                ..Default::default()
            });
        assert!(error.is_err(), "{kind} accepted an oversized target");
        assert_eq!(row(&files, result.entity_id), before);
        let after = owner_revision(&files, document);
        assert_eq!((&after.1, after.2), (&history.1, history.2));
    }
}
#[test]
fn deleting_reopened_blend_removes_its_dimensions_and_restores_them_on_undo() {
    for kind in ["fillet", "chamfer"] {
        let (mut files, document, mut source) = fresh();
        let result = create_corner(&mut files, &mut source, kind, "");
        let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
        let count_before = Client::new(FileQueryHost::new(&files))
            .sketch()
            .dimensions(&Default::default())
            .unwrap()
            .dimensions
            .unwrap()
            .len();
        assert!(count_before > 0);
        let sketch = owned_geometry(&files, reopened).sketches()[0].id();
        files
            .commands(reopened)
            .unwrap()
            .run("Delete blend", |snapshot, _| {
                snapshot.delete_sketch_entities(sketch, &[SketchEntityId(result.entity_id)])
            })
            .unwrap();
        assert!(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .dimensions(&Default::default())
                .unwrap()
                .dimensions
                .unwrap_or_default()
                .is_empty()
        );
        assert_eq!(
            rows(&files).iter().filter(|row| row.kind == "line").count(),
            2
        );
        files.commands(reopened).unwrap().undo().unwrap();
        assert_eq!(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .dimensions(&Default::default())
                .unwrap()
                .dimensions
                .unwrap()
                .len(),
            count_before
        );
        files.commands(reopened).unwrap().redo().unwrap();
        assert_eq!(
            rows(&files).iter().filter(|row| row.kind == "line").count(),
            2
        );
    }
}
#[test]
fn fixed_inferred_source_lines_accept_a_corner_without_new_redundant_dimensions() {
    for kind in ["fillet", "chamfer"] {
        let (mut files, _, mut source) = fresh();
        let first = add_call(
            &mut files,
            &mut source,
            &literal_input("line", &[[-5., 0.], [0., 0.]]),
        )
        .unwrap();
        let next = add_call(
            &mut files,
            &mut source,
            &literal_input("line", &[[0., 0.], [0., 5.]]),
        )
        .unwrap();
        for point in [
            first.point_i_ds.as_ref().unwrap()[0],
            next.point_i_ds.as_ref().unwrap()[1],
        ] {
            Client::new(FileHost::new(&mut files, &mut source, "test"))
                .constrain()
                .add(&oblikovati_api::wire::AddConstraintArgs {
                    kind: "fix".into(),
                    entities: Some(vec![point]),
                    ..Default::default()
                })
                .unwrap();
        }
        let request = AddSketchEntityArgs {
            kind: kind.into(),
            entity_refs: Some(vec![first.entity_id, next.entity_id]),
            radius: "width".into(),
            ..Default::default()
        };
        Client::new(
            FileHost::new(&mut files, &mut source, "test")
                .with_sketch_corner_construction(&MartensiteSketchConstruction, Default::default()),
        )
        .sketch()
        .add_entity(&request)
        .unwrap();
    }
}
#[test]
fn invalid_corner_creation_preserves_both_lines_and_history() {
    for (kind, radius) in [
        ("fillet", "-1 cm"),
        ("chamfer", "missing"),
        ("chamfer", "10 cm"),
    ] {
        let (mut files, document, mut source) = fresh();
        let first = add_call(
            &mut files,
            &mut source,
            &literal_input("line", &[[-5., 0.], [0., 0.]]),
        )
        .unwrap();
        let second = add_call(
            &mut files,
            &mut source,
            &literal_input("line", &[[0., 0.], [0., 5.]]),
        )
        .unwrap();
        let before = rows(&files);
        let history = owner_revision(&files, document);
        let input = AddSketchEntityArgs {
            kind: kind.into(),
            radius: radius.into(),
            entity_refs: Some(vec![first.entity_id, second.entity_id]),
            ..Default::default()
        };
        let error = Client::new(
            FileHost::new(&mut files, &mut source, "test")
                .with_sketch_corner_construction(&MartensiteSketchConstruction, Default::default()),
        )
        .sketch()
        .add_entity(&input);
        assert!(error.is_err());
        assert_eq!(rows(&files), before);
        let after = owner_revision(&files, document);
        assert_eq!((&after.1, after.2), (&history.1, history.2));
    }
}
#[test]
fn fillet_keeps_its_tangent_branch_for_line_order_and_endpoint_directions() {
    for swap in [false, true] {
        for reverse_first in [false, true] {
            for reverse_second in [false, true] {
                let (mut files, _, mut source) = fresh();
                Client::new(FileHost::new(&mut files, &mut source, "test"))
                    .sketch()
                    .set_inference_options(&oblikovati_api::wire::InferenceOptionsView {
                        infer_enabled: false,
                        constrain_enabled: false,
                        priority: "none".into(),
                    })
                    .unwrap();
                let mut a = [[-5., 0.], [0., 0.]];
                let mut b = [[0., 0.], [0., 5.]];
                if reverse_first {
                    a.reverse();
                }
                if reverse_second {
                    b.reverse();
                }
                let first = add_call(&mut files, &mut source, &literal_input("line", &a)).unwrap();
                let next = add_call(&mut files, &mut source, &literal_input("line", &b)).unwrap();
                let mut refs = vec![first.entity_id, next.entity_id];
                if swap {
                    refs.reverse();
                }
                let request = AddSketchEntityArgs {
                    kind: "fillet".into(),
                    entity_refs: Some(refs),
                    radius: "width".into(),
                    ..Default::default()
                };
                let result = Client::new(
                    FileHost::new(&mut files, &mut source, "test").with_sketch_corner_construction(
                        &MartensiteSketchConstruction,
                        Default::default(),
                    ),
                )
                .sketch()
                .add_entity(&request)
                .unwrap();
                set_width(&mut files, &mut source, "2 cm");
                assert!((row(&files, result.entity_id).radius - 2.).abs() < 1e-7);
            }
        }
    }
}
#[test]
fn explicit_associative_corner_method_uses_the_same_driving_path() {
    let (mut files, _, mut source) = fresh();
    let first = add_call(
        &mut files,
        &mut source,
        &literal_input("line", &[[-5., 0.], [0., 0.]]),
    )
    .unwrap();
    let second = add_call(
        &mut files,
        &mut source,
        &literal_input("line", &[[0., 0.], [0., 5.]]),
    )
    .unwrap();
    let request = AddSketchEntityArgs {
        kind: "fillet".into(),
        entity_refs: Some(vec![first.entity_id, second.entity_id]),
        radius: "width".into(),
        ..Default::default()
    };
    let result = Client::new(
        FileHost::new(&mut files, &mut source, "test")
            .with_sketch_corner_construction(&MartensiteSketchConstruction, Default::default()),
    )
    .sketch()
    .add_associative_entity(&request)
    .unwrap();
    set_width(&mut files, &mut source, "2 cm");
    assert!((row(&files, result.entity_id).radius - 2.).abs() < 1e-7);
}
#[test]
fn a_sick_unrelated_sketch_keeps_its_failure_local_to_its_owner() {
    let (mut files, document, mut source) = fresh();
    let corner = create_corner(&mut files, &mut source, "fillet", "");
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "other".into(),
            expression: "1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    create_empty_sketch(&mut files, &mut source);
    let mut circle = circle_input();
    circle.sketch_index = 1;
    circle.radius = "other".into();
    add_call(&mut files, &mut source, &circle).unwrap();
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "other".into(),
            expression: "-1 cm".into(),
            ..Default::default()
        });
    assert!(
        result.is_ok(),
        "unrelated sick sketch must not reject the edit: {result:?}"
    );
    assert!((row(&files, corner.entity_id).radius - 1.).abs() < 1e-7);
    assert_eq!(owned_geometry(&files, document).sketches().len(), 2);
}
#[test]
fn a_sick_unrelated_dimension_in_the_corner_sketch_remains_failure_local() {
    let (mut files, _, mut source) = fresh();
    create_corner(&mut files, &mut source, "fillet", "");
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "other".into(),
            expression: "1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let mut circle = circle_input();
    circle.radius = "other".into();
    add_call(&mut files, &mut source, &circle).unwrap();
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "other".into(),
            expression: "-1 cm".into(),
            ..Default::default()
        });
    assert!(
        result.is_ok(),
        "unrelated sick dimension must not reject the edit: {result:?}"
    );
}
#[test]
fn source_line_parameter_cannot_shrink_beneath_the_owned_setback() {
    let (mut files, document, mut source) = fresh();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "extent".into(),
            expression: "5 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let first = add_call(
        &mut files,
        &mut source,
        &literal_input("line", &[[-5., 0.], [0., 0.]]),
    )
    .unwrap();
    let second = add_call(
        &mut files,
        &mut source,
        &literal_input("line", &[[0., 0.], [0., 5.]]),
    )
    .unwrap();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .dimension()
        .add(&oblikovati_api::wire::AddDimensionArgs {
            kind: "distance".into(),
            entities: first.point_i_ds.clone(),
            expression: "extent".into(),
            ..Default::default()
        })
        .unwrap();
    let request = AddSketchEntityArgs {
        kind: "chamfer".into(),
        entity_refs: Some(vec![first.entity_id, second.entity_id]),
        radius: "width".into(),
        ..Default::default()
    };
    let blend = Client::new(
        FileHost::new(&mut files, &mut source, "test")
            .with_sketch_corner_construction(&MartensiteSketchConstruction, Default::default()),
    )
    .sketch()
    .add_entity(&request)
    .unwrap();
    let before = row(&files, blend.entity_id);
    let history = owner_revision(&files, document);
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "extent".into(),
            expression: "0.25 cm".into(),
            ..Default::default()
        });
    assert!(
        result.is_err(),
        "source shrink must reject the impossible setback: {result:?}"
    );
    assert_eq!(row(&files, blend.entity_id), before);
    let after = owner_revision(&files, document);
    assert_eq!((&after.1, after.2), (&history.1, history.2));
}
