// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::feature::*;

fn reader(id: u64, keys: &[FeatureReadKey], unknown: bool) -> AuthoredFeature<i64> {
    AuthoredFeature::new(
        FeatureId(id),
        "Reader",
        0,
        FeatureReadFootprint::new(keys.iter().copied(), unknown),
    )
}

fn read_program() -> FeatureSequence<i64> {
    let mut sequence = FeatureSequence::default();
    sequence.append(reader(1, &[], false)).unwrap();
    sequence
        .append(reader(2, &[FeatureReadKey::Parameter(1)], false))
        .unwrap();
    sequence
        .append(reader(
            3,
            &[
                FeatureReadKey::Sketch(1),
                FeatureReadKey::Feature(FeatureId(2)),
            ],
            false,
        ))
        .unwrap();
    sequence
}

#[test]
fn read_namespaces_and_deduplication_are_explicit() {
    let footprint = FeatureReadFootprint::new(
        [
            FeatureReadKey::Parameter(1),
            FeatureReadKey::Parameter(1),
            FeatureReadKey::Sketch(1),
            FeatureReadKey::Feature(FeatureId(1)),
        ],
        false,
    );
    assert_eq!(footprint.keys().len(), 3);
    assert!(!footprint.has_unattributed_reads());
    assert_ne!(footprint, FeatureReadFootprint::default());
}

#[test]
fn attributed_changes_replay_the_suffix_after_the_last_clean_prefix() {
    let sequence = read_program();
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(1)]),
        Some(FeatureReplayRange { start: 1, end: 3 })
    );
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Sketch(1)]),
        Some(FeatureReplayRange { start: 2, end: 3 })
    );
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(9)]),
        None
    );
}

#[test]
fn feature_edits_and_multiple_changed_reads_choose_the_earliest_start() {
    let sequence = read_program();
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Feature(FeatureId(1))]),
        Some(FeatureReplayRange { start: 0, end: 3 })
    );
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Sketch(1), FeatureReadKey::Parameter(1)]),
        Some(FeatureReplayRange { start: 1, end: 3 })
    );
}

#[test]
fn unattributed_reads_are_conservative_but_an_empty_change_is_not_an_edit() {
    let mut sequence = read_program();
    sequence.append(reader(4, &[], true)).unwrap();
    let feature = sequence.get(FeatureId(4)).unwrap();
    assert!(feature.reads().has_unattributed_reads());
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(9)]),
        Some(FeatureReplayRange { start: 3, end: 4 })
    );
    assert_eq!(sequence.replay_for_reads([]), None);
    sequence.set_end_of_part(Some(3)).unwrap();
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(9)]),
        None
    );
}

#[test]
fn cutoff_excludes_trailing_readers_but_suppression_keeps_active_dependencies() {
    let mut sequence = read_program();
    sequence.set_suppressed(FeatureId(2), true).unwrap();
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(1)]),
        Some(FeatureReplayRange { start: 1, end: 3 })
    );
    sequence.set_end_of_part(Some(2)).unwrap();
    assert_eq!(sequence.replay_for_reads([FeatureReadKey::Sketch(1)]), None);
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(1)]),
        Some(FeatureReplayRange { start: 1, end: 2 })
    );
}

#[test]
fn float_recipe_replacement_does_not_discard_signed_zero() {
    let mut sequence = FeatureSequence::default();
    sequence
        .append(AuthoredFeature::new(
            FeatureId(1),
            "Offset",
            0.0_f64,
            FeatureReadFootprint::default(),
        ))
        .unwrap();
    sequence
        .replace(FeatureId(1), -0.0, FeatureReadFootprint::default())
        .unwrap();
    assert_eq!(
        sequence.get(FeatureId(1)).unwrap().definition().to_bits(),
        (-0.0_f64).to_bits()
    );
}
