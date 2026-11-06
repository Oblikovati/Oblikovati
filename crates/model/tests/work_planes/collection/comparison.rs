// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn definition_comparison_retains_signed_zero_source_bits() {
    let plus = origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Constant(0.0));
    let minus = origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Constant(-0.0));
    assert!(!plus.same_recipe(&minus));
    assert!(plus.same_recipe(&plus));
    let first = WorkPlaneDefinition::Fixed(SketchPlane::xy());
    assert!(first.same_recipe(&first));
    assert!(!first.same_recipe(&WorkPlaneDefinition::Fixed(SketchPlane::yz())));
    assert!(!first.same_recipe(&plus));
    assert!(!plus.same_recipe(&origin_offset(
        WorkPlaneOrigin::XY,
        WorkPlaneDistance::Parameter(1)
    )));
}

#[test]
fn parameter_distances_compare_by_stable_identity() {
    let first = origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Parameter(1));
    assert!(first.same_recipe(&first));
    assert!(!first.same_recipe(&origin_offset(
        WorkPlaneOrigin::XY,
        WorkPlaneDistance::Parameter(2)
    )));
    assert!(!first.same_recipe(&origin_offset(
        WorkPlaneOrigin::XZ,
        WorkPlaneDistance::Parameter(1)
    )));
}

#[test]
fn program_round_trip_preserves_signed_zero_in_definitions_and_conditions() {
    let mut sequence = FeatureSequence::default();
    add(
        &mut sequence,
        1,
        origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Constant(-0.0)),
    );
    let condition = length_condition(9, FeatureSuppressionComparison::Equal, -0.0);
    sequence
        .set_suppression_condition(FeatureId(1), Some(condition))
        .unwrap();
    let restored = restored(&sequence);
    assert!(sequence.same_work_plane_recipe(&restored));
    let condition = restored.features()[0].suppression_condition().unwrap();
    assert_eq!(condition.threshold().value.to_bits(), (-0.0_f64).to_bits());
}

#[test]
fn changed_condition_comparison_and_threshold_are_authored_differences() {
    let sequence = authored_program();
    for (comparison, value) in [
        (FeatureSuppressionComparison::GreaterThan, 20.0),
        (FeatureSuppressionComparison::LessThan, 21.0),
    ] {
        let mut candidate = sequence.clone();
        let condition = length_condition(1, comparison, value);
        candidate
            .set_suppression_condition(FeatureId(3), Some(condition))
            .unwrap();
        assert!(!sequence.same_work_plane_recipe(&candidate));
    }
}

#[test]
fn changed_stable_feature_identity_is_an_authored_difference() {
    let sequence = authored_program();
    let mut recipe = sequence.work_plane_recipe().unwrap();
    recipe.features[2].id = FeatureId(9);
    let candidate = FeatureSequence::from_work_plane_recipe(&recipe).unwrap();
    assert!(!sequence.same_work_plane_recipe(&candidate));
}

#[test]
fn metadata_changes_are_authored_differences() {
    let sequence = authored_program();
    let mut renamed = sequence.clone();
    renamed.rename(FeatureId(1), "Renamed").unwrap();
    assert!(!sequence.same_work_plane_recipe(&renamed));
    let mut suppressed = sequence.clone();
    suppressed.set_suppressed(FeatureId(1), true).unwrap();
    assert!(!sequence.same_work_plane_recipe(&suppressed));
    let mut cutoff = sequence.clone();
    cutoff.set_end_of_part(None).unwrap();
    assert!(!sequence.same_work_plane_recipe(&cutoff));
}

#[test]
fn conditions_order_and_declared_reads_are_authored_differences() {
    let sequence = authored_program();
    let mut condition = sequence.clone();
    condition
        .set_suppression_condition(FeatureId(3), None)
        .unwrap();
    assert!(!sequence.same_work_plane_recipe(&condition));
    let mut order = sequence.clone();
    order.reorder(FeatureId(3), 0).unwrap();
    assert!(!sequence.same_work_plane_recipe(&order));
    let mut reads = sequence.clone();
    reads
        .replace(
            FeatureId(2),
            referenced_offset(1, -2.0),
            FeatureReadFootprint::new([FeatureReadKey::Feature(FeatureId(1))], false),
        )
        .unwrap();
    assert!(!sequence.same_work_plane_recipe(&reads));
}
