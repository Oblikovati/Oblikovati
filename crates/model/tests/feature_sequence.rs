// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{dependency::DependencyError, feature::*};
use std::error::Error;

fn plane(id: u64, keys: &[FeatureReadKey]) -> AuthoredFeature<i64> {
    AuthoredFeature::new(
        FeatureId(id),
        "Plane",
        id as i64,
        FeatureReadFootprint::new(keys.iter().copied(), false),
    )
}

fn program() -> FeatureSequence<i64> {
    let mut sequence = FeatureSequence::default();
    for id in 1..=3 {
        sequence.append(plane(id, &[])).unwrap();
    }
    sequence
}

fn ids(sequence: &FeatureSequence<i64>) -> Vec<u64> {
    sequence
        .features()
        .iter()
        .map(|feature| feature.id().0)
        .collect()
}

fn range(start: usize, end: usize) -> FeatureReplayRange {
    FeatureReplayRange { start, end }
}

#[test]
fn owned_recipe_accessors_and_names_do_not_change_identity() {
    let mut sequence = program();
    sequence.rename(FeatureId(2), "Base").unwrap();
    let feature = sequence.get(FeatureId(2)).unwrap();
    assert_eq!(feature.name(), "Base");
    assert_eq!(feature.definition().clone(), 2);
    assert!(!feature.suppressed());
    assert!(feature.reads().keys().is_empty());
    assert_eq!(ids(&sequence), vec![1, 2, 3]);
    assert_eq!(sequence.end_of_part(), None);
}

#[test]
fn valid_reorder_preserves_ids_and_replays_earliest_changed_position() {
    let mut sequence = program();
    assert_eq!(
        sequence.reorder(FeatureId(3), 0).unwrap(),
        Some(range(0, 3))
    );
    assert_eq!(ids(&sequence), vec![3, 1, 2]);
    assert_eq!(sequence.reorder(FeatureId(3), 0).unwrap(), None);
    assert_eq!(
        sequence.reorder(FeatureId(1), 2).unwrap(),
        Some(range(1, 3))
    );
    assert_eq!(ids(&sequence), vec![3, 2, 1]);
}

fn dependent_program() -> FeatureSequence<i64> {
    let mut sequence = program();
    sequence
        .replace(
            FeatureId(2),
            20,
            FeatureReadFootprint::new([FeatureReadKey::Feature(FeatureId(1))], false),
        )
        .unwrap();
    sequence
}

#[test]
fn reorder_cannot_cross_a_driver_in_either_direction() {
    let mut sequence = dependent_program();
    for (id, index) in [(2, 0), (1, 2)] {
        assert_eq!(
            sequence.reorder(FeatureId(id), index),
            Err(FeatureSequenceError::ForwardRead {
                feature: FeatureId(2),
                driver: FeatureId(1)
            })
        );
        assert_eq!(ids(&sequence), vec![1, 2, 3]);
    }
    sequence.reorder(FeatureId(3), 0).unwrap();
    assert_eq!(ids(&sequence), vec![3, 1, 2]);
}

#[test]
fn removal_protects_readers_then_truncates_the_old_result() {
    let mut sequence = dependent_program();
    assert_eq!(
        sequence.remove(FeatureId(1)),
        Err(FeatureSequenceError::Dependency(DependencyError::InUse {
            id: FeatureId(1),
            readers: vec![FeatureId(2)]
        }))
    );
    assert_eq!(sequence.remove(FeatureId(3)).unwrap(), range(2, 2));
    assert_eq!(sequence.remove(FeatureId(2)).unwrap(), range(1, 1));
    assert_eq!(sequence.remove(FeatureId(1)).unwrap(), range(0, 0));
    assert!(sequence.features().is_empty());
}

#[test]
fn explicit_suppression_retains_reads_and_invalidates_the_running_tail() {
    let mut sequence = dependent_program();
    assert_eq!(
        sequence.set_suppressed(FeatureId(2), true).unwrap(),
        Some(range(1, 3))
    );
    assert_eq!(sequence.set_suppressed(FeatureId(2), true).unwrap(), None);
    assert!(sequence.get(FeatureId(2)).unwrap().suppressed());
    assert!(sequence.remove(FeatureId(1)).is_err());
    assert_eq!(
        sequence.set_suppressed(FeatureId(2), false).unwrap(),
        Some(range(1, 3))
    );
}

#[test]
fn end_of_part_is_an_exclusive_positional_boundary() {
    let mut sequence = program();
    assert_eq!(
        sequence.set_end_of_part(Some(2)).unwrap(),
        Some(range(0, 2))
    );
    assert_eq!(sequence.set_end_of_part(Some(2)).unwrap(), None);
    sequence.reorder(FeatureId(3), 0).unwrap();
    assert_eq!(sequence.effective_end(), 2);
    assert_eq!(ids(&sequence)[..2], [3, 1]);
    assert_eq!(
        sequence.set_end_of_part(Some(0)).unwrap(),
        Some(range(0, 0))
    );
    assert_eq!(sequence.set_end_of_part(None).unwrap(), Some(range(0, 3)));
}

#[test]
fn explicit_boundary_survives_append_and_clamps_after_removal() {
    let mut sequence = program();
    sequence.set_end_of_part(Some(3)).unwrap();
    assert_eq!(sequence.append(plane(4, &[])).unwrap(), range(3, 3));
    assert_eq!(sequence.remove(FeatureId(4)).unwrap(), range(3, 3));
    sequence.remove(FeatureId(3)).unwrap();
    assert_eq!(sequence.effective_end(), 2);
    assert_eq!(sequence.end_of_part(), Some(3));
    assert_eq!(sequence.append(plane(5, &[])).unwrap(), range(2, 3));
}

#[test]
fn empty_prefix_can_be_restored_without_a_first_feature() {
    let mut sequence: FeatureSequence<i64> = FeatureSequence::default();
    assert_eq!(
        sequence.set_end_of_part(Some(0)).unwrap(),
        Some(range(0, 0))
    );
    sequence.append(plane(1, &[])).unwrap();
    assert_eq!(sequence.effective_end(), 0);
    assert_eq!(sequence.set_end_of_part(None).unwrap(), Some(range(0, 1)));
}

#[test]
fn duplicate_and_missing_drivers_do_not_install_a_feature() {
    let mut sequence = program();
    assert!(matches!(
        sequence.append(plane(1, &[])),
        Err(FeatureSequenceError::Dependency(
            DependencyError::Duplicate(FeatureId(1))
        ))
    ));
    assert!(matches!(
        sequence.append(plane(4, &[FeatureReadKey::Feature(FeatureId(9))])),
        Err(FeatureSequenceError::Dependency(DependencyError::Missing(
            FeatureId(9)
        )))
    ));
    assert_eq!(ids(&sequence), vec![1, 2, 3]);
}

#[test]
fn a_self_driver_does_not_reserve_the_candidate_id() {
    let mut sequence = program();
    assert!(matches!(
        sequence.append(plane(4, &[FeatureReadKey::Feature(FeatureId(4))])),
        Err(FeatureSequenceError::Dependency(
            DependencyError::Cycle { .. }
        ))
    ));
    assert_eq!(ids(&sequence), vec![1, 2, 3]);
    sequence
        .append(plane(4, &[FeatureReadKey::Feature(FeatureId(2))]))
        .unwrap();
}

#[test]
fn invalid_positions_leave_the_recipe_intact() {
    let mut sequence = program();
    assert_eq!(
        sequence.reorder(FeatureId(1), 3),
        Err(FeatureSequenceError::Index {
            index: 3,
            length: 3
        })
    );
    assert_eq!(ids(&sequence), vec![1, 2, 3]);
    assert_eq!(sequence.end_of_part(), None);
}

#[test]
fn unknown_ids_cannot_edit_the_recipe() {
    let mut sequence = program();
    assert_eq!(
        sequence.set_end_of_part(Some(4)),
        Err(FeatureSequenceError::Boundary {
            index: 4,
            length: 3
        })
    );
    assert!(sequence.rename(FeatureId(9), "Absent").is_err());
    assert!(sequence.remove(FeatureId(9)).is_err());
    assert!(sequence.set_suppressed(FeatureId(9), true).is_err());
    assert!(
        sequence
            .replace(FeatureId(9), 42, FeatureReadFootprint::default())
            .is_err()
    );
    assert_eq!(ids(&sequence), vec![1, 2, 3]);
    assert_eq!(sequence.end_of_part(), None);
}

#[test]
fn replacement_keeps_identity_name_and_suppression() {
    let mut sequence = dependent_program();
    sequence.rename(FeatureId(2), "Offset").unwrap();
    sequence.set_suppressed(FeatureId(2), true).unwrap();
    assert_eq!(
        sequence
            .replace(FeatureId(2), 99, FeatureReadFootprint::default())
            .unwrap(),
        range(1, 3)
    );
    let feature = sequence.get(FeatureId(2)).unwrap();
    assert_eq!(feature.name(), "Offset");
    assert!(feature.suppressed());
    assert_eq!(feature.definition().clone(), 99);
    sequence.remove(FeatureId(1)).unwrap();
}

#[test]
fn invalid_replacement_preserves_both_old_definition_and_edges() {
    let mut sequence = dependent_program();
    let forward = FeatureReadFootprint::new([FeatureReadKey::Feature(FeatureId(3))], false);
    assert_eq!(
        sequence.replace(FeatureId(2), 99, forward),
        Err(FeatureSequenceError::ForwardRead {
            feature: FeatureId(2),
            driver: FeatureId(3)
        })
    );
    assert_eq!(*sequence.get(FeatureId(2)).unwrap().definition(), 20);
    assert!(sequence.remove(FeatureId(1)).is_err());
    sequence.remove(FeatureId(3)).unwrap();
}

#[test]
fn cyclic_replacement_preserves_both_definition_and_edges() {
    let mut sequence = dependent_program();
    let cyclic = FeatureReadFootprint::new([FeatureReadKey::Feature(FeatureId(2))], false);
    assert!(matches!(
        sequence.replace(FeatureId(1), 99, cyclic),
        Err(FeatureSequenceError::Dependency(
            DependencyError::Cycle { .. }
        ))
    ));
    assert_eq!(*sequence.get(FeatureId(2)).unwrap().definition(), 20);
    assert!(sequence.remove(FeatureId(1)).is_err());
    sequence.remove(FeatureId(3)).unwrap();
}

#[test]
fn sequence_clone_has_independent_owned_definition_and_graph() {
    let original = dependent_program();
    let mut candidate = original.clone();
    candidate
        .replace(FeatureId(2), 99, FeatureReadFootprint::default())
        .unwrap();
    candidate.remove(FeatureId(1)).unwrap();
    assert_eq!(ids(&original), vec![1, 2, 3]);
    assert_eq!(*original.get(FeatureId(2)).unwrap().definition(), 20);
    assert_eq!(ids(&candidate), vec![2, 3]);
}

#[test]
fn errors_report_offending_values_expected_shape_and_dependency_source() {
    let index = FeatureSequenceError::Index {
        index: 5,
        length: 3,
    };
    assert_eq!(index.to_string(), "feature index 5: expected [0,3)");
    assert!(index.source().is_none());
    assert_eq!(
        FeatureSequenceError::Boundary {
            index: 5,
            length: 3
        }
        .to_string(),
        "end-of-part index 5: expected [0,3]"
    );
}

#[test]
fn dependency_errors_retain_their_cause_and_offending_ids() {
    assert!(
        FeatureSequenceError::ForwardRead {
            feature: FeatureId(2),
            driver: FeatureId(3)
        }
        .to_string()
        .contains("expected driver before reader")
    );
    let missing: FeatureSequenceError = DependencyError::Missing(FeatureId(8)).into();
    assert!(missing.to_string().contains("FeatureId(8)"));
    assert_eq!(missing.source().unwrap().to_string(), missing.to_string());
}
