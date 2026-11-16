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
}
#[test]
fn projected_source_cannot_move_inside_the_owned_setback() {
    use oblikovati_model::{
        document_sketch::SketchPointProjection,
        feature::{
            AuthoredFeature,
            work_geometry::{DatumLength, WorkGeometryDefinition, WorkPointReference},
        },
    };
    let (mut files, document, mut source) = fresh();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .set_inference_options(&oblikovati_api::wire::InferenceOptionsView {
            infer_enabled: false,
            constrain_enabled: false,
            priority: "none".into(),
        })
        .unwrap();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "projected_x".into(),
            expression: "-5 cm".into(),
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
    for point in [
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
    let far = SketchEntityId(first.point_i_ds.as_ref().unwrap()[0]);
    let sketch = owned_geometry(&files, document).sketches()[0].id();
    files
        .commands(document)
        .unwrap()
        .run("Project corner source", |snapshot, ids| {
            let driver = snapshot.parameters.find("projected_x").unwrap();
            let feature = ids.allocate_feature().map_err(|e| e.to_string())?;
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(
                    feature,
                    "Projected source",
                    WorkGeometryDefinition::Point([
                        DatumLength::Parameter(driver),
                        DatumLength::Constant(0.),
                        DatumLength::Constant(0.),
                    ]),
                ))
                .map_err(|e| e.to_string())?;
            let mut bindings = snapshot
                .sketches
                .get(sketch)
                .unwrap()
                .point_projections()
                .to_vec();
            bindings.push(SketchPointProjection {
                anchor: far,
                reference: WorkPointReference::Feature(feature),
            });
            snapshot
                .sketches
                .set_point_projections(sketch, bindings)
                .map_err(|e| e.to_string())
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
            name: "projected_x".into(),
            expression: "-0.25 cm".into(),
            ..Default::default()
        });
    assert!(
        result.is_err(),
        "projected source must not move inside the setback: {result:?}"
    );
    assert_eq!(row(&files, blend.entity_id), before);
    let after = owner_revision(&files, document);
    assert_eq!((&after.1, after.2), (&history.1, history.2));
}
