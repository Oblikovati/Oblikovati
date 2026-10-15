// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::scene_fixture as fixture;
use oblikovati_model::{
    feature::{AuthoredFeature, FeatureId, work_geometry::DatumLength},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;
fn stored_bytes(independent: bool) -> Vec<u8> {
    let mut snapshot = fixture::selected_prism_document();
    if independent {
        append_independent_body(&mut snapshot);
    }
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    let parameters = package
        .with_parameters(&snapshot.parameters, UnitsOfMeasure::default())
        .unwrap();
    let sketches = parameters
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap();
    sketches
        .with_work_geometry(&snapshot.work_geometry, if independent { 2 } else { 1 })
        .unwrap()
        .encode()
        .unwrap()
}
fn append_independent_body(snapshot: &mut oblikovati_model::history::DocumentSnapshot) {
    let mut recipe = fixture::selected_prism_definition().recipe();
    recipe.displacement[2] = DatumLength::Constant(2.);
    let definition =
        oblikovati_model::feature::sketch_prism::SketchPrismDefinition::from_recipe(&recipe)
            .unwrap();
    let feature =
        AuthoredFeature::sketch_prism(FeatureId(2), "Independent", definition, snapshot).unwrap();
    snapshot.work_geometry.append(feature).unwrap();
}
fn open_prism(owner: &mut TestOwner, independent: bool) -> u64 {
    let path = std::env::temp_dir().join("head-scene.opd");
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), stored_bytes(independent));
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id
}
fn set_parameter(owner: &mut TestOwner, name: &str, expression: &str) {
    Client::new(owner.document_host())
        .parameters()
        .set(&ParameterSetArgs {
            name: name.into(),
            expression: expression.into(),
            ..Default::default()
        })
        .unwrap();
}
#[test]
fn desktop_publishes_current_generations_and_clears_after_close() {
    let (mut owner, _) = owner();
    assert!(owner.take_scene().unwrap().is_none());
    let document = open_prism(&mut owner, false);
    let first = owner.take_scene().unwrap().unwrap();
    assert_eq!(first.status.visible_bodies, 1);
    assert!(!first.scene.triangles.is_empty());
    assert!(owner.take_scene().unwrap().is_none());
    set_parameter(&mut owner, "width", "6 cm");
    let edited = owner.take_scene().unwrap().unwrap();
    assert!(edited.status.source.unwrap().1 > first.status.source.unwrap().1);
    verify_closed_owner_scene(&mut owner, document);
}
#[test]
fn failed_feature_is_visible_in_status_and_independent_body_is_presented() {
    let (mut owner, _) = owner();
    open_prism(&mut owner, true);
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        2
    );
    set_parameter(&mut owner, "depth", "0 cm");
    let update = owner.take_scene().unwrap().unwrap();
    assert_eq!(update.status.visible_bodies, 1);
    assert!(update.status.title().contains("1 unavailable features"));
    assert!(!update.scene.triangles.is_empty());
    assert!(matches!(
        update.status.health[0].status,
        oblikovati_model::feature::FeatureEvaluationStatus::Failed(_)
    ));
}

fn verify_closed_owner_scene(owner: &mut TestOwner, document: u64) {
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    let closed = owner.take_scene().unwrap().unwrap();
    assert!(closed.scene.triangles.is_empty());
    assert_eq!(closed.status.title(), "Oblikovati");
}

fn published_keys(owner: &TestOwner, document: u64, feature: FeatureId) -> Vec<Vec<u8>> {
    let workspace = owner.documents().workspace();
    let guid = workspace.identity(document).unwrap().internal_name;
    let session = workspace.session(document).unwrap();
    owner
        .part_scenes
        .current(session)
        .unwrap()
        .body(feature)
        .unwrap()
        .reference_keys(session.history(), guid)
        .unwrap()
}
#[test]
fn displayed_bodies_publish_current_names_in_distinct_feature_namespaces() {
    use oblikovati_model::identity::ReferenceEnvelope;
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    owner.take_scene().unwrap().unwrap();
    let first = published_keys(&owner, document, FeatureId(1));
    let second = published_keys(&owner, document, FeatureId(2));
    assert_eq!((first.len(), second.len()), (7, 7));
    assert!(
        first
            .iter()
            .all(|key| ReferenceEnvelope::decode(key).unwrap().context == 1)
    );
    assert!(
        second
            .iter()
            .all(|key| ReferenceEnvelope::decode(key).unwrap().context == 2)
    );
    assert!(!first.iter().any(|key| second.contains(key)));
    set_parameter(&mut owner, "width", "6 cm");
    let session = owner.documents().workspace().session(document).unwrap();
    assert!(owner.part_scenes.current(session).is_err());
    owner.take_scene().unwrap().unwrap();
    assert_eq!(published_keys(&owner, document, FeatureId(1)), first);
    assert_eq!(published_keys(&owner, document, FeatureId(2)), second);
    verify_published_face(&owner, document);
    owner.files.commands(document).unwrap().undo().unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_eq!(published_keys(&owner, document, FeatureId(1)), first);
    owner.files.commands(document).unwrap().redo().unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_eq!(published_keys(&owner, document, FeatureId(1)), first);
    verify_published_face(&owner, document);
}
fn verify_published_face(owner: &TestOwner, document: u64) {
    use oblikovati_model::identity::SemanticReferenceName;
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let key = SemanticReferenceName::new(guid, 1, 1, "sketch-prism", "start", &[1])
        .unwrap()
        .encode()
        .unwrap();
    let evaluated = owner.part_scenes.current(session).unwrap();
    let face = evaluated
        .body(FeatureId(1))
        .unwrap()
        .face_report(session.history(), guid, &key)
        .unwrap();
    assert!((face.area - 18.).abs() < 1e-10);
    assert!(
        evaluated
            .body(FeatureId(2))
            .unwrap()
            .face_report(session.history(), guid, &key)
            .is_err()
    );
}
#[test]
fn failed_replay_removes_exact_names_while_independent_names_remain_current() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    owner.take_scene().unwrap().unwrap();
    let retained = published_keys(&owner, document, FeatureId(2));
    set_parameter(&mut owner, "depth", "0 cm");
    owner.take_scene().unwrap().unwrap();
    let session = owner.documents().workspace().session(document).unwrap();
    assert!(
        owner
            .part_scenes
            .current(session)
            .unwrap()
            .body(FeatureId(1))
            .is_none()
    );
    assert_eq!(published_keys(&owner, document, FeatureId(2)), retained);
}

#[path = "part_scene/assignments.rs"]
mod assignments;
#[path = "part_scene/bodies.rs"]
mod bodies;
#[path = "part_scene/part_mass.rs"]
mod part_mass;
#[path = "part_scene/physical.rs"]
mod physical;
#[test]
fn typed_kernel_decline_reaches_desktop_status_and_recovers() {
    use oblikovati_martensite::{EvaluationEntity, EvaluationFailureKind, EvaluationPhase};
    use oblikovati_model::feature::{FeatureEvaluationStatus, part::PartFeatureFailure};
    let (mut owner, _) = owner();
    open_prism(&mut owner, false);
    owner.take_scene().unwrap().unwrap();
    set_parameter(&mut owner, "width", "0.000000000000001 cm");
    let update = owner.take_scene().unwrap().unwrap();
    let FeatureEvaluationStatus::Failed(PartFeatureFailure::Kernel(cause)) =
        &update.status.health[0].status
    else {
        panic!(
            "thin profile: expected adapter failure, got {:?}",
            update.status.health
        );
    };
    assert_eq!(cause.phase(), EvaluationPhase::Profile);
    assert_eq!(cause.kind(), EvaluationFailureKind::InvalidGeometry);
    assert_eq!(cause.entity(), &EvaluationEntity::Edge(101));
    assert_eq!(cause.feature(), Some(FeatureId(1)));
    assert_eq!(update.status.visible_bodies, 0);
    assert!(update.scene.triangles.is_empty());
    set_parameter(&mut owner, "width", "4 cm");
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}

#[path = "part_scene/cube.rs"]
mod cube;
#[path = "part_scene/sketches.rs"]
mod sketches;
