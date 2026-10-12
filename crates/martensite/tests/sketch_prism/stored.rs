// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::retain_stored_sketch_prism;
use oblikovati_model::{
    feature::{FeatureId, sketch_prism::SketchPrismDefinition, work_geometry::DatumLength},
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::RecipeIdAllocator,
};
fn stored_owner() -> DocumentEditHistory {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    let mut owner = DocumentEditHistory::new(document, ids).unwrap();
    let token = owner.begin("Extrude inputs").unwrap();
    let definition = stored_definition();
    owner
        .create_sketch_prism(&token, "Extrude", definition)
        .unwrap();
    owner.commit(&token, false).unwrap();
    owner
}
fn stored_definition() -> SketchPrismDefinition {
    SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
}
fn assert_stored_volume(owner: &DocumentEditHistory, expected: f64) {
    let body = retain_stored_sketch_prism(owner.snapshot(), FeatureId(1)).unwrap();
    assert_volume(body.report().properties.volume, expected);
}
#[test]
fn owned_stored_inputs_build_and_restore_kernel_results_through_undo() {
    let mut owner = stored_owner();
    let first = retain_stored_sketch_prism(owner.snapshot(), FeatureId(1)).unwrap();
    assert_volume(first.report().properties.volume, 24.);
    let token = owner.begin("Depth").unwrap();
    owner
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "30 mm")
        })
        .unwrap();
    owner.commit(&token, false).unwrap();
    assert_stored_volume(&owner, 36.);
    owner.undo().unwrap();
    assert_stored_volume(&owner, 24.);
    assert_volume(first.report().properties.volume, 24.);
}
#[test]
fn stored_suppressed_cutoff_and_absent_entries_do_not_build_shapes() {
    let owner = stored_owner();
    let mut snapshot: DocumentSnapshot = owner.snapshot().clone();
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), true)
        .unwrap();
    assert!(retain_stored_sketch_prism(&snapshot, FeatureId(1)).is_err());
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), false)
        .unwrap();
    snapshot.work_geometry.set_end_of_part(Some(0)).unwrap();
    assert!(retain_stored_sketch_prism(&snapshot, FeatureId(1)).is_err());
    assert!(retain_stored_sketch_prism(&snapshot, FeatureId(99)).is_err());
}

fn conditioned_snapshot(threshold: f64) -> DocumentSnapshot {
    use oblikovati_model::{
        feature::{FeatureSuppressionComparison, FeatureSuppressionCondition},
        quantity::{Dimension, Quantity},
    };
    let mut snapshot = stored_owner().snapshot().clone();
    let condition = FeatureSuppressionCondition::new(
        3,
        FeatureSuppressionComparison::GreaterThan,
        Quantity::new(threshold, Dimension::Length),
    )
    .unwrap();
    snapshot
        .work_geometry
        .set_suppression_condition(FeatureId(1), Some(condition))
        .unwrap();
    snapshot
}
fn assert_stored_decline(snapshot: &DocumentSnapshot, expected: &str) {
    let error = retain_stored_sketch_prism(snapshot, FeatureId(1))
        .err()
        .unwrap();
    assert!(error.contains(expected), "{error}");
}
#[test]
fn stored_condition_selects_enabled_or_suppressed_inputs() {
    let enabled = conditioned_snapshot(3.);
    let body = retain_stored_sketch_prism(&enabled, FeatureId(1)).unwrap();
    assert_volume(body.report().properties.volume, 24.);
    assert_stored_decline(&conditioned_snapshot(1.), "conditional suppression=true");
}
#[test]
fn stored_condition_failure_has_no_last_good_result_and_manual_suppression_wins() {
    let mut failed = conditioned_snapshot(3.);
    failed
        .parameters
        .set_expression(3, "missing_depth")
        .unwrap();
    assert_stored_decline(&failed, "suppression");
    failed
        .work_geometry
        .set_suppressed(FeatureId(1), true)
        .unwrap();
    assert_stored_decline(&failed, "suppressed=true");
}
#[test]
fn stored_datum_is_declined_as_a_non_body_definition() {
    use oblikovati_model::{
        feature::{AuthoredFeature, work_plane::WorkPlaneDefinition},
        sketch::SketchPlane,
    };
    let mut snapshot = stored_owner().snapshot().clone();
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_plane(
            FeatureId(2),
            "Datum",
            WorkPlaneDefinition::Fixed(SketchPlane::xy()),
        ))
        .unwrap();
    let error = retain_stored_sketch_prism(&snapshot, FeatureId(2))
        .err()
        .unwrap();
    assert!(error.contains("expected stored sketch prism inputs"));
}
