// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_VALUE_UPPER, USER_PARAM};
use oblikovati_model::{
    feature::{work_plane::*, *},
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    sketch::SketchPlane,
    units::UnitsOfMeasure,
};
use std::error::Error;

mod work_planes {
    use super::*;
    mod collection;
    mod recipe;
}

struct UnavailablePlanes;
impl WorkPlaneResolver for UnavailablePlanes {
    fn plane(&self, feature: FeatureId) -> Result<SketchPlane, WorkPlaneError> {
        Err(WorkPlaneError::Unavailable {
            feature,
            expected: "plane",
        })
    }
}

fn parameters(value: f64, dimension: Dimension) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_value(1, "gap", USER_PARAM, Quantity::new(value, dimension))
        .unwrap();
    parameters
}

fn offset(base: WorkPlaneReference, distance: WorkPlaneDistance) -> WorkPlaneDefinition {
    WorkPlaneDefinition::Offset { base, distance }
}

fn origin_offset(origin: WorkPlaneOrigin, distance: WorkPlaneDistance) -> WorkPlaneDefinition {
    offset(WorkPlaneReference::Origin(origin), distance)
}

fn add(
    sequence: &mut FeatureSequence<WorkPlaneDefinition>,
    id: u64,
    definition: WorkPlaneDefinition,
) {
    sequence
        .append(AuthoredFeature::work_plane(
            FeatureId(id),
            "Plane",
            definition,
        ))
        .unwrap();
}

fn referenced_offset(base: u64, value: f64) -> WorkPlaneDefinition {
    let reference = WorkPlaneReference::Feature(FeatureId(base));
    offset(reference, WorkPlaneDistance::Constant(value))
}

fn assert_origin(report: &WorkPlaneEvaluation, id: u64, expected: [f64; 3]) {
    let plane = report.plane(FeatureId(id)).unwrap();
    assert_eq!(plane.recipe().origin, expected);
}

fn failed_parameters() -> ParameterCollection<u64> {
    let mut parameters = parameters(5.0, Dimension::Length);
    parameters.set_expression(1, "missing").unwrap();
    parameters
}

fn chain() -> FeatureSequence<WorkPlaneDefinition> {
    let mut sequence = FeatureSequence::default();
    add(
        &mut sequence,
        1,
        origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Parameter(1)),
    );
    add(&mut sequence, 2, referenced_offset(1, -2.0));
    add(
        &mut sequence,
        3,
        WorkPlaneDefinition::Fixed(SketchPlane::yz()),
    );
    sequence
}

#[test]
fn offsets_follow_origin_normals_and_preserve_axes() {
    let parameters = parameters(5.0, Dimension::Length);
    for (origin, expected) in [
        (WorkPlaneOrigin::XY, [0.0, 0.0, 5.0]),
        (WorkPlaneOrigin::XZ, [0.0, -5.0, 0.0]),
        (WorkPlaneOrigin::YZ, [5.0, 0.0, 0.0]),
    ] {
        let definition = origin_offset(origin, WorkPlaneDistance::Parameter(1));
        let result = definition
            .evaluate(&parameters, &UnavailablePlanes)
            .unwrap();
        assert_eq!(result.recipe().origin, expected);
        assert_eq!(result.recipe().x_axis, origin.plane().recipe().x_axis);
        assert_eq!(result.recipe().y_axis, origin.plane().recipe().y_axis);
    }
}

#[test]
fn parameter_changes_and_tolerances_move_chained_current_frames() {
    let sequence = chain();
    let mut parameters = parameters(5.0, Dimension::Length);
    assert_origin(
        &sequence.evaluate_work_planes(&parameters).unwrap(),
        2,
        [0.0, 0.0, 3.0],
    );
    parameters
        .set_value(1, Quantity::new(12.0, Dimension::Length))
        .unwrap();
    let report = sequence.evaluate_work_planes(&parameters).unwrap();
    assert_origin(&report, 2, [0.0, 0.0, 10.0]);
}

#[test]
fn tolerance_selection_moves_chained_current_frames() {
    let sequence = chain();
    let mut parameters = parameters(12.0, Dimension::Length);
    parameters.set_symmetric(1, 1.0).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    let report = sequence.evaluate_work_planes(&parameters).unwrap();
    assert_origin(&report, 1, [0.0, 0.0, 13.0]);
    assert_origin(&report, 2, [0.0, 0.0, 11.0]);
    assert!(
        report
            .health
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy)
    );
}

#[test]
fn custom_frame_offsets_preserve_authored_orientation() {
    let frame = SketchPlane::new(
        [7.0, 8.0, 9.0],
        [0.6, 0.8, 0.0],
        [0.0, 0.0, 1.0],
        Default::default(),
    )
    .unwrap();
    let mut sequence = FeatureSequence::default();
    add(&mut sequence, 1, WorkPlaneDefinition::Fixed(frame));
    add(&mut sequence, 2, referenced_offset(1, 5.0));
    let report = sequence
        .evaluate_work_planes(&parameters(0.0, Dimension::Length))
        .unwrap();
    let shifted = report.plane(FeatureId(2)).unwrap();
    assert_eq!(shifted.recipe().origin, [11.0, 5.0, 9.0]);
    assert_eq!(shifted.recipe().x_axis, frame.recipe().x_axis);
    assert_eq!(shifted.options(), frame.options());
}

#[test]
fn failed_parameter_quarantines_readers_and_keeps_independent_geometry() {
    let sequence = chain();
    let report = sequence.evaluate_work_planes(&failed_parameters()).unwrap();
    assert!(matches!(
        report.health[0].status,
        FeatureEvaluationStatus::Failed(WorkPlaneError::Parameter(_))
    ));
    assert_eq!(
        report.health[1].status,
        FeatureEvaluationStatus::Blocked {
            drivers: vec![FeatureId(1)]
        }
    );
    assert!(report.plane(FeatureId(1)).is_err());
    assert!(report.plane(FeatureId(2)).is_err());
    assert_eq!(report.plane(FeatureId(3)).unwrap(), SketchPlane::yz());
}

#[test]
fn repaired_parameter_restores_chained_current_geometry() {
    let sequence = chain();
    let mut parameters = failed_parameters();
    assert!(
        sequence
            .evaluate_work_planes(&parameters)
            .unwrap()
            .plane(FeatureId(2))
            .is_err()
    );
    parameters
        .set_value(1, Quantity::new(9.0, Dimension::Length))
        .unwrap();
    let report = sequence.evaluate_work_planes(&parameters).unwrap();
    assert_origin(&report, 2, [0.0, 0.0, 7.0]);
}

#[test]
fn suppressed_source_planes_have_no_current_geometry_for_readers() {
    let mut sequence = chain();
    sequence.set_suppressed(FeatureId(1), true).unwrap();
    let report = sequence
        .evaluate_work_planes(&parameters(5.0, Dimension::Length))
        .unwrap();
    assert_eq!(report.health[0].status, FeatureEvaluationStatus::Suppressed);
    assert_eq!(
        report.health[1].status,
        FeatureEvaluationStatus::Failed(WorkPlaneError::Unavailable {
            feature: FeatureId(1),
            expected: "plane"
        })
    );
    assert!(report.plane(FeatureId(2)).is_err());
    assert!(report.plane(FeatureId(3)).is_ok());
}

#[test]
fn rolled_back_planes_are_unavailable() {
    let mut sequence = chain();
    sequence.set_end_of_part(Some(1)).unwrap();
    let report = sequence
        .evaluate_work_planes(&parameters(5.0, Dimension::Length))
        .unwrap();
    assert_eq!(report.health.len(), 1);
    let error = report.plane(FeatureId(2)).unwrap_err();
    assert!(
        error
            .to_string()
            .contains("FeatureId(2): expected an available current plane")
    );
    assert!(error.source().is_none());
}

#[test]
fn definitions_and_entries_share_one_read_footprint() {
    let sequence = chain();
    let feature = &sequence.features()[0];
    assert_eq!(feature.reads(), &feature.definition().reads());
    let readers: Vec<_> = sequence.features()[1]
        .reads()
        .keys()
        .iter()
        .copied()
        .collect();
    assert_eq!(readers, vec![FeatureReadKey::Feature(FeatureId(1))]);
}

#[test]
fn unknown_plane_sources_are_refused_and_unavailable_resolution_is_named() {
    let definition = referenced_offset(9, 1.0);
    let mut sequence = FeatureSequence::default();
    assert!(
        sequence
            .append(AuthoredFeature::work_plane(
                FeatureId(1),
                "Lost",
                definition
            ))
            .is_err()
    );
    assert_eq!(
        definition.evaluate(&parameters(1.0, Dimension::Length), &UnavailablePlanes),
        Err(WorkPlaneError::Unavailable {
            feature: FeatureId(9),
            expected: "plane"
        })
    );
}

#[test]
fn undeclared_definition_reads_cannot_start_replay() {
    let sequence = undeclared_program();
    let error = sequence
        .evaluate_work_planes(&parameters(5.0, Dimension::Length))
        .err()
        .unwrap();
    let expected = FeatureReplayError::Footprint {
        feature: FeatureId(1),
        missing: vec![FeatureReadKey::Parameter(1)],
    };
    assert_eq!(error, expected);
    assert!(
        error
            .to_string()
            .contains("missing reads [Parameter(1)]: expected every definition read")
    );
}

fn undeclared_program() -> FeatureSequence<WorkPlaneDefinition> {
    let definition = origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Parameter(1));
    let mut sequence = FeatureSequence::default();
    sequence
        .append(AuthoredFeature::new(
            FeatureId(1),
            "Bad",
            definition,
            FeatureReadFootprint::default(),
        ))
        .unwrap();
    sequence
}

#[test]
fn offsets_refuse_wrong_dimensions_nonfinite_values_and_overflow() {
    let definition = origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Parameter(1));
    let error = definition
        .evaluate(&parameters(1.0, Dimension::Angle), &UnavailablePlanes)
        .unwrap_err();
    assert!(error.to_string().contains("Some(1) value Quantity"));
    assert!(error.to_string().contains("expected finite Length"));
    for value in [f64::NAN, f64::INFINITY] {
        let definition = origin_offset(WorkPlaneOrigin::XY, WorkPlaneDistance::Constant(value));
        assert!(matches!(
            definition.evaluate(&parameters(1.0, Dimension::Length), &UnavailablePlanes),
            Err(WorkPlaneError::Offset { .. })
        ));
    }
}

#[test]
fn coordinate_overflow_is_a_frame_error_and_has_its_source() {
    let frame = overflowing_frame();
    let mut sequence = FeatureSequence::default();
    add(&mut sequence, 1, WorkPlaneDefinition::Fixed(frame));
    add(&mut sequence, 2, referenced_offset(1, f64::MAX));
    let report = sequence
        .evaluate_work_planes(&parameters(0.0, Dimension::Length))
        .unwrap();
    let FeatureEvaluationStatus::Failed(error) = &report.health[1].status else {
        panic!("expected a failed overflowing frame");
    };
    assert!(matches!(error, WorkPlaneError::Frame(_)));
    assert!(error.source().is_some());
    assert!(error.to_string().contains("inf"));
    assert!(report.plane(FeatureId(2)).is_err());
}

fn overflowing_frame() -> SketchPlane {
    let mut recipe = SketchPlane::yz().recipe();
    recipe.origin[0] = f64::MAX;
    SketchPlane::from_recipe(recipe, Default::default()).unwrap()
}

#[test]
fn failed_conditional_frame_is_not_adopted_and_quarantines_plane_readers() {
    let mut sequence = chain();
    let threshold = Quantity::new(2.0, Dimension::Length);
    let condition =
        FeatureSuppressionCondition::new(9, FeatureSuppressionComparison::LessThan, threshold)
            .unwrap();
    sequence
        .set_suppression_condition(FeatureId(1), Some(condition))
        .unwrap();
    let report = sequence
        .evaluate_work_planes(&parameters(5.0, Dimension::Length))
        .unwrap();
    assert!(matches!(
        report.health[0].status,
        FeatureEvaluationStatus::ConditionFailed(_)
    ));
    assert!(report.plane(FeatureId(1)).is_err());
    assert!(report.plane(FeatureId(2)).is_err());
    assert!(report.plane(FeatureId(3)).is_ok());
}
