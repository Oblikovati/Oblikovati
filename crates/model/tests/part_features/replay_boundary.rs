// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{FeatureReadFootprint, FeatureReplayRange};

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
fn input_changes_force_the_complete_active_program() {
    let old = document_with_two_planes();
    let mut edited = old.clone();
    edited.parameters.set_expression(3, "3 cm").unwrap();
    assert_eq!(
        old.part_replay_change(&edited),
        Some(FeatureReplayRange { start: 0, end: 2 })
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
