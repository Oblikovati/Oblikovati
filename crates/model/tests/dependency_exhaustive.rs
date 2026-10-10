// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::dependency::DependencyGraph;

// Independent matrix oracle: enumerate every forward-edge graph of four nodes.
// Graph insertion order opposes numeric order to expose accidental key sorting.
fn forward_edges(mask: u8) -> [[bool; 4]; 4] {
    let mut edges = [[false; 4]; 4];
    let mut bit = 0;
    for (reader, row) in edges.iter_mut().enumerate() {
        for edge in row.iter_mut().take(reader) {
            *edge = mask & (1 << bit) != 0;
            bit += 1;
        }
    }
    edges
}

fn matrix_closure(mut edges: [[bool; 4]; 4]) -> [[bool; 4]; 4] {
    for pivot in 0..4 {
        let pivot_row = edges[pivot];
        for row in &mut edges {
            extend_reachability(row, &pivot_row, pivot);
        }
    }
    edges
}

fn extend_reachability(row: &mut [bool; 4], pivot_row: &[bool; 4], pivot: usize) {
    let reaches_pivot = row[pivot];
    for (entry, pivot_entry) in row.iter_mut().zip(pivot_row) {
        *entry |= reaches_pivot && *pivot_entry;
    }
}

fn graph_from_matrix(edges: &[[bool; 4]; 4]) -> DependencyGraph<usize> {
    let mut graph = DependencyGraph::default();
    for id in (0..4).rev() {
        graph.insert(id).unwrap();
    }
    for (reader, row) in edges.iter().enumerate() {
        let drivers: Vec<usize> = (0..4).filter(|driver| row[*driver]).collect();
        graph.replace_drivers(reader, &drivers).unwrap();
    }
    graph
}

fn assert_order(graph: &DependencyGraph<usize>, closure: &[[bool; 4]; 4], changed: usize) {
    let order = graph.recompute_order(changed).unwrap();
    let affected: Vec<usize> = (0..4)
        .filter(|id| *id == changed || closure[*id][changed])
        .collect();
    let mut sorted = order.clone();
    sorted.sort();
    assert_eq!(sorted, affected);
    for (index, reader) in order.iter().enumerate() {
        assert!(
            order[index + 1..]
                .iter()
                .all(|driver| !closure[*reader][*driver])
        );
    }
}

fn assert_edits(graph: &DependencyGraph<usize>, closure: &[[bool; 4]; 4], reader: usize) {
    for (driver, reachable) in closure.iter().enumerate() {
        let mut edited = graph.clone();
        let expected_cycle = reader == driver || reachable[reader];
        assert_eq!(
            edited.replace_drivers(reader, &[driver]).is_err(),
            expected_cycle
        );
    }
}

#[test]
fn every_four_node_forward_graph_matches_independent_reachability() {
    for mask in 0..64 {
        let edges = forward_edges(mask);
        let closure = matrix_closure(edges);
        let graph = graph_from_matrix(&edges);
        for changed in 0..4 {
            assert_order(&graph, &closure, changed);
            assert_edits(&graph, &closure, changed);
        }
    }
}
