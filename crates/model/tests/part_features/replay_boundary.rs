// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{FeatureReadFootprint, FeatureReplayRange};

#[test]
fn associative_sketch_inputs_require_ordered_part_recompute() {
    use oblikovati_model::{
        document_sketch::{SketchOrientationDefinition, SketchPointProjection},
        feature::{work_axis::WorkAxisReference, work_geometry::WorkPointReference},
        sketch::{SketchEntityId, SketchFrameAxis},
    };
    let source = selected_prism_document();
    assert!(!source.sketches.requires_part_dependency_recompute());
    let mut placed = source.clone();
    placed
        .sketches
        .attach_work_plane(SKETCH, FeatureId(7))
        .unwrap();
    assert!(placed.sketches.requires_part_dependency_recompute());
    let mut projected = source.clone();
    projected
        .sketches
        .set_point_projections(
            SKETCH,
            vec![SketchPointProjection {
                anchor: SketchEntityId(1),
                reference: WorkPointReference::Origin,
            }],
        )
        .unwrap();
    assert!(projected.sketches.requires_part_dependency_recompute());
    let mut oriented = source;
    oriented
        .sketches
        .set_orientation(
            SKETCH,
            Some(SketchOrientationDefinition {
                reference: WorkAxisReference::OriginX,
                sketch_axis: SketchFrameAxis::X,
                reverse: false,
                origin: None,
            }),
        )
        .unwrap();
    assert!(oriented.sketches.requires_part_dependency_recompute());
}

fn document_with_two_planes() -> DocumentSnapshot {
    let mut document = part_document();
    document
        .work_geometry
        .append(AuthoredFeature::work_plane(
            FeatureId(2),
            "Second",
            WorkPlaneDefinition::Fixed(SketchPlane::xy()),
        ))
        .unwrap();
    document
}

#[test]
fn finds_recipe_and_cutoff_changes_without_losing_the_clean_prefix() {
    let old = document_with_two_planes();
    assert_eq!(old.part_replay_change(&old.clone()), None);
    let mut edited = old.clone();
    edited
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    assert_eq!(
        old.part_replay_change(&edited),
        Some(FeatureReplayRange { start: 1, end: 2 })
    );
    edited.work_geometry.set_end_of_part(Some(1)).unwrap();
    assert_eq!(
        old.part_replay_change(&edited),
        Some(FeatureReplayRange { start: 1, end: 1 })
    );
}

#[test]
fn unused_parameter_change_keeps_features_but_sketch_edits_restart() {
    let old = document_with_two_planes();
    let mut edited = old.clone();
    edited.parameters.set_expression(3, "3 cm").unwrap();
    assert_eq!(
        old.part_replay_change(&edited),
        Some(FeatureReplayRange { start: 2, end: 2 })
    );
    let mut sketch_edit = old.clone();
    sketch_edit.sketches.set_visible(SKETCH, false).unwrap();
    assert_eq!(
        old.part_replay_change(&sketch_edit),
        Some(FeatureReplayRange { start: 0, end: 2 })
    );
}

#[test]
fn unattributed_prefix_reads_prevent_tail_only_replay() {
    let mut old = document_with_two_planes();
    let definition = old
        .work_geometry
        .get(FeatureId(1))
        .unwrap()
        .definition()
        .clone();
    old.work_geometry
        .replace(
            FeatureId(1),
            definition,
            FeatureReadFootprint::new([], true),
        )
        .unwrap();
    let mut edited = old.clone();
    edited
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    assert_eq!(
        old.part_replay_change(&edited),
        Some(FeatureReplayRange { start: 0, end: 2 })
    );
}

#[test]
fn a_changed_read_footprint_invalidates_its_feature() {
    let old = document_with_two_planes();
    let mut edited = old.clone();
    let definition = edited
        .work_geometry
        .get(FeatureId(1))
        .unwrap()
        .definition()
        .clone();
    edited
        .work_geometry
        .replace(
            FeatureId(1),
            definition,
            FeatureReadFootprint::new([], true),
        )
        .unwrap();
    assert_eq!(
        old.part_replay_change(&edited),
        Some(FeatureReplayRange { start: 0, end: 2 })
    );
    assert!(
        !old.work_geometry
            .same_work_geometry_recipe(&edited.work_geometry)
    );
}

fn append_parameter_and_point(document: &mut DocumentSnapshot) {
    document
        .parameters
        .insert_expression(4, "new_length", oblikovati_api::types::USER_PARAM, "1 cm")
        .unwrap();
    let point = PartFeatureDefinition::Point([
        DatumLength::Parameter(4),
        DatumLength::Constant(0.),
        DatumLength::Constant(0.),
    ]);
    document
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(3), "New point", point))
        .unwrap();
}

#[test]
fn new_tail_parameter_does_not_invalidate_an_unrelated_prefix() {
    let source = document_with_two_planes();
    let mut candidate = source.clone();
    append_parameter_and_point(&mut candidate);
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 2, end: 3 })
    );
    candidate.parameters.set_expression(3, "3 cm").unwrap();
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 0, end: 3 })
    );
}

#[test]
fn added_parameter_restarts_when_an_existing_feature_already_reads_it() {
    let mut source = part_document();
    let point = PartFeatureDefinition::Point([
        DatumLength::Parameter(4),
        DatumLength::Constant(0.),
        DatumLength::Constant(0.),
    ]);
    source
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(2), "Missing input", point))
        .unwrap();
    let mut candidate = source.clone();
    append_parameter_and_point(&mut candidate);
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 0, end: 3 })
    );
}

#[test]
fn added_parameter_without_an_appended_feature_restarts_the_program() {
    let source = document_with_two_planes();
    let mut candidate = source.clone();
    candidate
        .parameters
        .insert_expression(4, "new_length", oblikovati_api::types::USER_PARAM, "1 cm")
        .unwrap();
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 0, end: 2 })
    );
}

#[test]
fn added_parameter_with_unknown_prefix_reads_restarts_the_program() {
    let mut source = document_with_two_planes();
    let definition = source
        .work_geometry
        .get(FeatureId(1))
        .unwrap()
        .definition()
        .clone();
    source
        .work_geometry
        .replace(
            FeatureId(1),
            definition,
            FeatureReadFootprint::new([], true),
        )
        .unwrap();
    let mut candidate = source.clone();
    append_parameter_and_point(&mut candidate);
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 0, end: 3 })
    );
}

fn append_parameter_point(document: &mut DocumentSnapshot, parameter: u64) {
    let point = PartFeatureDefinition::Point([
        DatumLength::Parameter(parameter),
        DatumLength::Constant(0.),
        DatumLength::Constant(0.),
    ]);
    document
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(3), "Point", point))
        .unwrap();
}

#[test]
fn parameter_change_starts_at_the_first_actual_reader() {
    let mut source = document_with_two_planes();
    append_parameter_point(&mut source, 3);
    let mut candidate = source.clone();
    candidate.parameters.set_expression(3, "3 cm").unwrap();
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 2, end: 3 })
    );
}

#[test]
fn derived_parameter_reader_is_dirty_when_its_driver_changes() {
    let mut source = document_with_two_planes();
    source
        .parameters
        .insert_expression(4, "derived", oblikovati_api::types::USER_PARAM, "depth * 2")
        .unwrap();
    append_parameter_point(&mut source, 4);
    let mut candidate = source.clone();
    candidate.parameters.set_expression(3, "3 cm").unwrap();
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 2, end: 3 })
    );
}

#[test]
fn current_definitions_recover_missing_stored_parameter_reads() {
    let mut source = document_with_two_planes();
    let point = PartFeatureDefinition::Point([
        DatumLength::Parameter(3),
        DatumLength::Constant(0.),
        DatumLength::Constant(0.),
    ]);
    source
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Point",
            point,
            FeatureReadFootprint::new([], false),
        ))
        .unwrap();
    let mut candidate = source.clone();
    candidate.parameters.set_expression(3, "3 cm").unwrap();
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 2, end: 3 })
    );
}

#[test]
fn current_sketch_dependencies_recover_missing_stored_reads() {
    let mut source = document_with_two_planes();
    let prism =
        AuthoredFeature::sketch_prism(FeatureId(3), "Prism", selected_prism_definition(), &source)
            .unwrap();
    source.work_geometry.append(prism).unwrap();
    let definition = source
        .work_geometry
        .get(FeatureId(3))
        .unwrap()
        .definition()
        .clone();
    source
        .work_geometry
        .replace(
            FeatureId(3),
            definition,
            FeatureReadFootprint::new([], false),
        )
        .unwrap();
    let mut candidate = source.clone();
    candidate.parameters.set_expression(1, "5 cm").unwrap();
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 2, end: 3 })
    );
}

#[test]
fn unknown_reads_force_full_replay_for_parameter_changes() {
    let mut source = document_with_two_planes();
    let definition = source
        .work_geometry
        .get(FeatureId(2))
        .unwrap()
        .definition()
        .clone();
    source
        .work_geometry
        .replace(
            FeatureId(2),
            definition,
            FeatureReadFootprint::new([], true),
        )
        .unwrap();
    let mut candidate = source.clone();
    candidate.parameters.set_expression(3, "3 cm").unwrap();
    assert_eq!(
        source.part_replay_change(&candidate),
        Some(FeatureReplayRange { start: 0, end: 2 })
    );
}
