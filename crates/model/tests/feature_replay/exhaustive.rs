// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use std::collections::BTreeSet;

const EDGES: [(u64, u64); 6] = [(1, 2), (1, 3), (1, 4), (2, 3), (2, 4), (3, 4)];

fn graph_program(mask: u8) -> FeatureSequence<u64> {
    let mut sequence = FeatureSequence::default();
    for id in 1..=4 {
        let drivers = EDGES
            .iter()
            .enumerate()
            .filter_map(|(bit, (driver, reader))| {
                (mask & (1 << bit) != 0 && *reader == id).then_some(*driver)
            })
            .collect::<Vec<_>>();
        sequence.append(authored(id, &drivers)).unwrap();
    }
    sequence
}

fn descendants(mask: u8, root: u64) -> BTreeSet<u64> {
    let mut affected = BTreeSet::from([root]);
    loop {
        let before = affected.len();
        for (bit, (driver, reader)) in EDGES.iter().enumerate() {
            if mask & (1 << bit) != 0 && affected.contains(driver) {
                affected.insert(*reader);
            }
        }
        if affected.len() == before {
            return affected;
        }
    }
}

fn fail_root(sequence: &FeatureSequence<u64>, root: u64) -> Vec<ReplayRecord> {
    let mut controller = full_replay(sequence);
    while let Some(action) = controller.advance().unwrap() {
        if let FeatureReplayStep::Evaluate(feature) = action {
            let outcome = if feature.0 == root {
                Err(ReplayFault::LostEntity(root))
            } else {
                Ok(())
            };
            controller.record_result(feature, outcome).unwrap();
        }
    }
    controller.finish().unwrap()
}

fn verify_graph(mask: u8, root: u64) {
    let sequence = graph_program(mask);
    let records = fail_root(&sequence, root);
    let expected = descendants(mask, root);
    let actual: BTreeSet<u64> = records
        .iter()
        .filter_map(|record| {
            matches!(
                record.status,
                FeatureEvaluationStatus::Failed(_) | FeatureEvaluationStatus::Blocked { .. }
            )
            .then_some(record.feature.0)
        })
        .collect();
    assert_eq!(actual, expected, "mask={mask}, root={root}");
    assert_eq!(records.len(), 4);
}

#[test]
fn quarantine_matches_graph_reachability_for_every_four_feature_graph_and_failure() {
    for mask in 0..64 {
        for root in 1..=4 {
            verify_graph(mask, root);
        }
    }
}
