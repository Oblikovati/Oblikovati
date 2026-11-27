// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::json::JsonValue;

#[test]
fn api_extrusion_respects_member_scope_and_updates_the_scene() {
    let (owner, awake) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = create(&mut owner, &awake);
    let primary = active_member(&owner, document);
    let variant = add_member(&mut owner);
    activate_member(&mut owner, variant);
    author_cube(&mut owner);
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_member_scene(&mut owner, document, primary, true, 0);
    assert_member_scene(&mut owner, document, variant, false, 1);
}

fn active_member(owner: &TestOwner, document: u64) -> u64 {
    owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .model_states
        .as_ref()
        .unwrap()
        .active()
        .unwrap()
        .id
}

fn add_member(owner: &mut TestOwner) -> u64 {
    Client::new(owner.document_host())
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Variant".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state
        .id
}

fn activate_member(owner: &mut TestOwner, member: u64) {
    Client::new(owner.document_host())
        .model_states()
        .activate(&RepRef { id: member })
        .unwrap();
}

fn assert_member_scene(
    owner: &mut TestOwner,
    document: u64,
    member: u64,
    suppressed: bool,
    count: usize,
) {
    activate_member(owner, member);
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let feature = snapshot.work_geometry.features()[0].id();
    assert_eq!(snapshot.feature_suppressed(feature).unwrap(), suppressed);
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        count
    );
}

#[test]
fn api_cube_rebuilds_through_history_storage_and_named_faces() {
    let (owner, awake) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = create(&mut owner, &awake);
    author_cube(&mut owner);
    assert_extrude_scalar(&mut owner, 1, 10.);
    reject_invalid_extrude_edit(&mut owner, 1);
    assert_extrude_scalar(&mut owner, 1, 10.);
    let original = current_cube(&mut owner);
    assert_volume(&mut owner, 1_000.);
    let source_keys = published_keys(&owner, document, oblikovati_model::feature::FeatureId(1));
    edit_extrude_distance(&mut owner, 1, "20 mm");
    assert_shared_size_unchanged(&owner, document);
    assert_extrude_scalar(&mut owner, 1, 20.);
    assert_eq!(
        current_cube(&mut owner).scene.triangles.len(),
        original.scene.triangles.len()
    );
    assert_volume(&mut owner, 2_000.);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_volume(&mut owner, 1_000.);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_volume(&mut owner, 2_000.);
    save_reopen_cube(&mut owner, document);
    assert_volume(&mut owner, 2_000.);
    assert_cube_topology(&mut owner);
    let reopened = owner.documents().workspace().active_document().unwrap();
    assert_eq!(
        published_keys(&owner, reopened, oblikovati_model::feature::FeatureId(1)),
        source_keys
    );
    assert_cube_face_properties(&owner);
}

fn assert_shared_size_unchanged(owner: &TestOwner, document: u64) {
    let session = owner.documents().workspace().session(document).unwrap();
    let parameters = &session.snapshot().parameters;
    let size = parameters.find("size").unwrap();
    assert_eq!(parameters.model_value(size).unwrap().value, 1.);
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
            new_full_document_name: document_fixture_path!("/memory/cube.opd").into(),
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
            full_document_name: document_fixture_path!("/memory/cube.opd").into(),
            ..Default::default()
        })
        .unwrap();
}

fn assert_cube_face_properties(owner: &TestOwner) {
    let document = owner.documents().workspace().active_document().unwrap();
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let body = owner
        .part_scenes
        .current(session)
        .unwrap()
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    let faces = published_keys(owner, document, oblikovati_model::feature::FeatureId(1))
        .into_iter()
        .filter(|key| face_key(key))
        .collect::<Vec<_>>();
    assert_eq!(faces.len(), 6);
    let mut areas = Vec::new();
    for key in faces {
        let face = body
            .as_prism()
            .unwrap()
            .face_report(session.history(), guid, &key)
            .unwrap();
        assert_eq!(face.loops, 1);
        areas.push((face.area * 1000.).round() as i64);
    }
    areas.sort_unstable();
    assert_eq!(areas, [1000, 1000, 2000, 2000, 2000, 2000]);
}

fn face_key(key: &[u8]) -> bool {
    oblikovati_model::identity::ReferenceEnvelope::decode(key)
        .is_ok_and(|reference| reference.kind == 1)
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
            .all(|face| face.kind == "plane" && !face.key.is_empty())
    );
}
