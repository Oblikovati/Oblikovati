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
