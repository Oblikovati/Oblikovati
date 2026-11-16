// SPDX-License-Identifier: GPL-2.0-only
use super::*;
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
            name: "projected_x".into(),
            expression: "-6 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let geometry = owned_geometry(&files, document).sketches()[0].geometry();
    assert!((geometry.point(far).unwrap().position[0] + 6.).abs() < 1e-7);
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "projected_x".into(),
            expression: "-0.25 cm".into(),
            ..Default::default()
        });
    assert!(
        result.is_err(),
        "partial solve must not hide an invalid projected corner"
    );
}
