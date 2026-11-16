// SPDX-License-Identifier: GPL-2.0-only
use super::*;
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
#[test]
fn pattern_driven_source_cannot_cross_the_owned_setback() {
    let (mut files, document, mut source) = fresh();
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    client
        .sketch()
        .set_inference_options(&oblikovati_api::wire::InferenceOptionsView {
            infer_enabled: false,
            constrain_enabled: false,
            priority: "none".into(),
        })
        .unwrap();
    client
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "spacing".into(),
            expression: "5 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let seed = add_call(
        &mut files,
        &mut source,
        &literal_input("point", &[[-10., 0.]]),
    )
    .unwrap();
    let pattern = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .pattern()
        .rectangular(&oblikovati_api::wire::AddSketchPatternArgs {
            kind: "rectangular".into(),
            entities: Some(vec![seed.entity_id]),
            count1: 2,
            count2: 1,
            spacing1: "spacing".into(),
            spacing2: "1 cm".into(),
            dir1: Some(vec![1., 0.]),
            dir2: Some(vec![0., 1.]),
            ..Default::default()
        })
        .unwrap();
    let member = pattern.created.unwrap()[0];
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
    for point in [
        seed.entity_id,
        first.point_i_ds.as_ref().unwrap()[1],
        second.point_i_ds.as_ref().unwrap()[1],
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
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .constrain()
        .add(&oblikovati_api::wire::AddConstraintArgs {
            kind: "coincident".into(),
            entities: Some(vec![member, first.point_i_ds.as_ref().unwrap()[0]]),
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
    let changed = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "spacing".into(),
            expression: "9.75 cm".into(),
            ..Default::default()
        });
    assert!(
        changed.is_err(),
        "pattern must not move the far point across the setback: {changed:?}"
    );
    assert_eq!(row(&files, blend.entity_id), before);
    let after = owner_revision(&files, document);
    assert_eq!((&after.1, after.2), (&history.1, history.2));
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
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "other".into(),
            expression: "-1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "spacing".into(),
            expression: "4 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let geometry = owned_geometry(&files, document).sketches()[0].geometry();
    let far = geometry
        .point(SketchEntityId(first.point_i_ds.as_ref().unwrap()[0]))
        .unwrap()
        .position;
    assert!((far[0] + 6.).abs() < 1e-7);
    assert_eq!(geometry.rectangular_patterns().len(), 1);
    let changed = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "spacing".into(),
            expression: "9.75 cm".into(),
            ..Default::default()
        });
    assert!(
        changed.is_err(),
        "partial solve must not hide an invalid pattern-driven corner"
    );
}
#[test]
fn healthy_corner_recomputes_while_a_disconnected_dimension_stays_sick() {
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
    let mut circle = circle_input();
    circle.radius = "other".into();
    let other = add_call(&mut files, &mut source, &circle).unwrap();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "other".into(),
            expression: "-1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "width".into(),
            expression: "2 cm".into(),
            ..Default::default()
        });
    assert!(
        result.is_ok(),
        "a sick disconnected dimension must not block a valid corner edit: {result:?}"
    );
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let geometry = snapshot.sketches.sketches()[0].geometry();
    assert!(
        (geometry
            .arc_radius(SketchEntityId(corner.entity_id))
            .unwrap()
            - 2.)
            .abs()
            < 1e-7
    );
    assert_eq!(
        geometry
            .circle(SketchEntityId(other.entity_id))
            .unwrap()
            .radius,
        1.
    );
    assert!(matches!(
        snapshot
            .document_sketch_health(snapshot.sketches.sketches()[0].id())
            .unwrap(),
        oblikovati_model::document_sketch::PlacedSketchHealth::Current(
            oblikovati_model::document_sketch::SketchEvaluation::Failed(_)
        )
    ));
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "width".into(),
            expression: "0 cm".into(),
            ..Default::default()
        });
    assert!(
        result.is_err(),
        "the component solve must not hide an invalid corner"
    );
    files.commands(document).unwrap().undo().unwrap();
    let geometry = owned_geometry(&files, document).sketches()[0].geometry();
    assert!(
        (geometry
            .arc_radius(SketchEntityId(corner.entity_id))
            .unwrap()
            - 1.)
            .abs()
            < 1e-7
    );
    files.commands(document).unwrap().redo().unwrap();
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    let geometry = owned_geometry(&files, reopened).sketches()[0].geometry();
    assert!(
        (geometry
            .arc_radius(SketchEntityId(corner.entity_id))
            .unwrap()
            - 2.)
            .abs()
            < 1e-7
    );
}

#[path = "dependencies/projected.rs"]
mod projected;
