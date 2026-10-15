// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::json::JsonValue;

#[test]
fn api_cube_rebuilds_through_history_storage_and_named_faces() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    author_cube(&mut owner);
    assert_extrude_scalar(&mut owner, 1, 10.);
    reject_invalid_extrude_edit(&mut owner, 1);
    assert_extrude_scalar(&mut owner, 1, 10.);
    let original = current_cube(&mut owner);
    assert_volume(&mut owner, 1_000.);
    edit_extrude_distance(&mut owner, 1, "20 mm");
    assert_extrude_scalar(&mut owner, 1, 20.);
    assert_eq!(
        current_cube(&mut owner).scene.triangles.len(),
        original.scene.triangles.len()
    );
    assert_volume(&mut owner, 8_000.);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_volume(&mut owner, 1_000.);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_volume(&mut owner, 8_000.);
    save_reopen_cube(&mut owner, document);
    assert_volume(&mut owner, 8_000.);
    assert_cube_topology(&mut owner);
}

fn author_cube(owner: &mut TestOwner) {
    add_size(owner);
    let sketch = create_xy_sketch(owner);
    let rectangle = add_rectangle(owner, sketch);
    constrain_rectangle(owner, sketch, &rectangle);
    add_extrude(owner, sketch);
}

fn add_size(owner: &mut TestOwner) {
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "size".into(),
            expression: "10 mm".into(),
            ..Default::default()
        })
        .unwrap();
}

fn create_xy_sketch(owner: &mut TestOwner) -> i64 {
    Client::new(owner.document_host())
        .sketch()
        .create(&CreateSketchArgs {
            plane: "XY".into(),
            ..Default::default()
        })
        .unwrap()
        .sketch_index
}

fn add_rectangle(owner: &mut TestOwner, sketch: i64) -> AddSketchEntityResult {
    Client::new(owner.document_host())
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            sketch_index: sketch,
            kind: "rectangle".into(),
            points: Some(vec![Some(vec![0., 0.]), Some(vec![1., 1.])]),
            ..Default::default()
        })
        .unwrap()
}

fn constrain_rectangle(owner: &mut TestOwner, sketch: i64, rectangle: &AddSketchEntityResult) {
    let lines = rectangle.entity_i_ds.as_deref().unwrap();
    let points = rectangle.point_i_ds.as_deref().unwrap();
    for (kind, entities) in [
        ("horizontal", vec![lines[0]]),
        ("vertical", vec![lines[1]]),
        ("horizontal", vec![lines[2]]),
        ("vertical", vec![lines[3]]),
        ("fix", vec![points[0]]),
    ] {
        add_constraint(owner, sketch, kind, entities);
    }
    add_dimension(owner, sketch, &[points[0], points[1]], "horizontal");
    add_dimension(owner, sketch, &[points[1], points[2]], "vertical");
    assert_eq!(sketch_status(owner, sketch).dof, 0);
}

fn add_constraint(owner: &mut TestOwner, sketch: i64, kind: &str, entities: Vec<u64>) {
    Client::new(owner.document_host())
        .constrain()
        .add(&AddConstraintArgs {
            sketch_index: sketch,
            kind: kind.into(),
            entities: Some(entities),
            ..Default::default()
        })
        .unwrap();
}

fn add_dimension(owner: &mut TestOwner, sketch: i64, entities: &[u64], orientation: &str) {
    Client::new(owner.document_host())
        .dimension()
        .add(&AddDimensionArgs {
            sketch_index: sketch,
            kind: "distance".into(),
            entities: Some(entities.to_vec()),
            expression: "size".into(),
            orientation: orientation.into(),
            ..Default::default()
        })
        .unwrap();
}

fn sketch_status(owner: &mut TestOwner, sketch: i64) -> ConstraintStatusResult {
    Client::new(owner.document_host())
        .sketch()
        .constraint_status(&SketchArgs {
            sketch_index: sketch,
        })
        .unwrap()
}

fn add_extrude(owner: &mut TestOwner, sketch: i64) {
    let args = serde_json::to_value(oblikovati_api::wire::featureargs::Extrude {
        sketch_index: sketch,
        profile_index: 0,
        distance: "size".into(),
        operation: "new".into(),
        ..Default::default()
    })
    .unwrap();
    Client::new(owner.document_host())
        .features()
        .add(&AddFeatureArgs {
            kind: "extrude".into(),
            args: JsonValue(Some(args)),
        })
        .unwrap();
}

fn reject_invalid_extrude_edit(owner: &mut TestOwner, id: u64) {
    let result = Client::new(owner.document_host())
        .features()
        .edit(&EditFeatureArgs {
            id,
            scalars: Some(vec![ScalarEdit {
                index: 1,
                value: "20 mm".into(),
            }]),
            ..Default::default()
        });
    assert!(result.is_err());
}

fn edit_extrude_distance(owner: &mut TestOwner, id: u64, value: &str) {
    Client::new(owner.document_host())
        .features()
        .edit(&EditFeatureArgs {
            id,
            scalars: Some(vec![ScalarEdit {
                index: 0,
                value: value.into(),
            }]),
            ..Default::default()
        })
        .unwrap();
}

fn assert_extrude_scalar(owner: &mut TestOwner, id: u64, expected: f64) {
    let feature = Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap()
        .feature;
    let scalar = &feature.scalars.unwrap()[0];
    assert_eq!(scalar.label, "Distance");
    assert_eq!(scalar.unit, "mm");
    assert_eq!(scalar.value, expected);
}

fn current_cube(owner: &mut TestOwner) -> crate::HeadSceneUpdate {
    owner.take_scene().unwrap().unwrap()
}

fn assert_volume(owner: &mut TestOwner, expected: f64) {
    let result = Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs::default())
        .unwrap();
    assert!(
        (result.volume_mm3 - expected).abs() < 1e-6,
        "volume {}, expected {expected}",
        result.volume_mm3
    );
}

fn save_reopen_cube(owner: &mut TestOwner, document: u64) {
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: "/memory/cube.opd".into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/memory/cube.opd".into(),
            ..Default::default()
        })
        .unwrap();
}

fn assert_cube_topology(owner: &mut TestOwner) {
    let keys = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    let bodies = keys.bodies.unwrap();
    let faces = bodies[0].faces.as_ref().unwrap();
    assert_eq!(faces.len(), 6);
    assert!(
        faces
            .iter()
            .all(|face| face.kind == "face" && !face.key.is_empty())
    );
}
