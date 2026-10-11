// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey,
        work_geometry::{
            DatumLength, WorkGeometryDefinition, WorkGeometryEvaluation, WorkPointReference,
        },
    },
    history::DocumentSnapshot,
    sketch::SketchPlane,
};
fn append(snapshot: &mut DocumentSnapshot, id: u64, definition: WorkGeometryDefinition) {
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(id), "Datum", definition))
        .unwrap();
}
fn associated(id: u64) -> WorkGeometryDefinition {
    WorkGeometryDefinition::PointAtReference(WorkPointReference::Feature(FeatureId(id)))
}
fn coordinates(position: [f64; 3]) -> WorkGeometryDefinition {
    WorkGeometryDefinition::Point(position.map(DatumLength::Constant))
}
fn snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    append(&mut snapshot, 1, coordinates([-0.0, 0.0, 4.0]));
    append(&mut snapshot, 2, associated(1));
    append(&mut snapshot, 3, associated(2));
    append(
        &mut snapshot,
        4,
        WorkGeometryDefinition::PointAtReference(WorkPointReference::Origin),
    );
    snapshot
}
fn current(snapshot: &DocumentSnapshot, id: u64) -> [f64; 3] {
    evaluation(snapshot).point(FeatureId(id)).unwrap()
}
#[test]
fn chained_associations_retain_exact_current_coordinates_and_direct_dependencies() {
    let mut snapshot = snapshot();
    assert_eq!(
        current(&snapshot, 3).map(f64::to_bits),
        [-0.0f64, 0.0, 4.0].map(f64::to_bits)
    );
    assert!(
        associated(1)
            .reads()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(1)))
    );
    let replacement = coordinates([1.0, 2.0, 3.0]);
    snapshot
        .work_geometry
        .replace(FeatureId(1), replacement, replacement.reads())
        .unwrap();
    assert_eq!(current(&snapshot, 3), [1.0, 2.0, 3.0]);
    assert_reference_identity();
}
#[test]
fn suppressed_and_cutoff_sources_produce_no_stale_associated_coordinates() {
    let mut snapshot = snapshot();
    assert_eq!(current(&snapshot, 3)[2], 4.0);
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), true)
        .unwrap();
    let failed = evaluation(&snapshot);
    assert!(failed.point(FeatureId(2)).is_err());
    assert!(failed.point(FeatureId(3)).is_err());
    assert_eq!(failed.point(FeatureId(4)).unwrap(), [0.0; 3]);
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), false)
        .unwrap();
    assert_eq!(current(&snapshot, 3)[2], 4.0);
    snapshot.work_geometry.set_end_of_part(Some(1)).unwrap();
    let cutoff = evaluation(&snapshot);
    assert!(cutoff.point(FeatureId(2)).is_err());
}
#[test]
fn wrong_kind_is_local_and_independent_associations_continue() {
    let mut snapshot = snapshot();
    append(
        &mut snapshot,
        5,
        WorkGeometryDefinition::Fixed(SketchPlane::xy()),
    );
    append(&mut snapshot, 6, associated(5));
    append(&mut snapshot, 7, associated(4));
    let evaluation = evaluation(&snapshot);
    assert!(evaluation.point(FeatureId(6)).is_err());
    let receipt = &evaluation.health[5].status;
    assert!(
        matches!(receipt, oblikovati_model::feature::FeatureEvaluationStatus::Failed(cause) if cause.to_string().contains("kind plane"))
    );
    assert_eq!(evaluation.point(FeatureId(7)).unwrap(), [0.0; 3]);
}
#[test]
fn associated_sources_round_trip_and_refuse_older_wrapper_and_program_versions() {
    let snapshot = snapshot();
    let mut recipe = snapshot.work_geometry.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 6);
    assert_eq!(recipe.features[1].definition.version, 4);
    let restored =
        oblikovati_model::feature::FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(snapshot.work_geometry.same_work_geometry_recipe(&restored));
    recipe.version = 5;
    assert!(
        oblikovati_model::feature::FeatureSequence::from_work_geometry_recipe(&recipe).is_err()
    );
    let mut definition = associated(1).recipe();
    definition.version = 3;
    assert!(WorkGeometryDefinition::from_recipe(definition).is_err());
}

fn evaluation(snapshot: &DocumentSnapshot) -> WorkGeometryEvaluation {
    snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap()
}

fn assert_reference_identity() {
    assert!(!associated(1).same_recipe(associated(2)));
    assert!(associated(1).same_recipe(associated(1)));
    assert!(!associated(1).same_recipe(coordinates([1.0, 2.0, 3.0])));
}
