// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::feature::*;

const ORDERED_EDGES: [(u64, u64); 6] = [(1, 2), (1, 3), (1, 4), (2, 3), (2, 4), (3, 4)];

fn selected_edges(mask: u8) -> Vec<(u64, u64)> {
    ORDERED_EDGES
        .iter()
        .enumerate()
        .filter_map(|(bit, edge)| (mask & (1 << bit) != 0).then_some(*edge))
        .collect()
}

fn ordered_program(edges: &[(u64, u64)]) -> FeatureSequence<u64> {
    let mut sequence = FeatureSequence::default();
    for id in 1..=4 {
        let reads = edges
            .iter()
            .filter(|(_, reader)| *reader == id)
            .map(|(driver, _)| FeatureReadKey::Feature(FeatureId(*driver)));
        sequence
            .append(AuthoredFeature::new(
                FeatureId(id),
                "Feature",
                id,
                FeatureReadFootprint::new(reads, false),
            ))
            .unwrap();
    }
    sequence
}

fn verify_move(edges: &[(u64, u64)], from: usize, to: usize) {
    let mut sequence = ordered_program(edges);
    let original = vec![1, 2, 3, 4];
    let mut expected = original.clone();
    let moved = expected.remove(from);
    expected.insert(to, moved);
    let valid = edges.iter().all(|(driver, reader)| {
        expected.iter().position(|id| id == driver) < expected.iter().position(|id| id == reader)
    });
    let result = sequence.reorder(FeatureId(moved), to);
    assert_eq!(result.is_ok(), valid, "edges={edges:?}, {from}->{to}");
    let actual: Vec<u64> = sequence
        .features()
        .iter()
        .map(|feature| feature.id().0)
        .collect();
    assert_eq!(actual, if valid { expected } else { original });
}

#[test]
fn every_four_feature_graph_move_preserves_exactly_the_dependency_valid_orders() {
    for mask in 0..64 {
        let edges = selected_edges(mask);
        for from in 0..4 {
            for to in 0..4 {
                verify_move(&edges, from, to);
            }
        }
    }
}
