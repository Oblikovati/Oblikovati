// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::dependency::{DependencyError, DependencyGraph};

fn registered(ids: &[u32]) -> DependencyGraph<u32> {
    let mut graph = DependencyGraph::default();
    for id in ids {
        graph.insert(*id).unwrap();
    }
    graph
}

#[test]
fn chain_recomputes_only_transitive_readers() {
    let mut graph = registered(&[1, 2, 3, 4]);
    graph.replace_drivers(2, &[1]).unwrap();
    graph.replace_drivers(3, &[2, 1]).unwrap();
    assert_eq!(graph.recompute_order(1).unwrap(), [1, 2, 3]);
    assert_eq!(graph.recompute_order(2).unwrap(), [2, 3]);
    assert_eq!(graph.recompute_order(4).unwrap(), [4]);
}

#[test]
fn diamond_uses_insertion_order_and_evaluates_join_once() {
    let mut graph = registered(&[90, 30, 20, 10]);
    graph.replace_drivers(30, &[90]).unwrap();
    graph.replace_drivers(20, &[90]).unwrap();
    graph.replace_drivers(10, &[20, 30, 20]).unwrap();
    assert_eq!(graph.drivers(10).unwrap(), [30, 20]);
    assert_eq!(graph.readers(90).unwrap(), [30, 20]);
    assert_eq!(graph.recompute_order(90).unwrap(), [90, 30, 20, 10]);
}

#[test]
fn edge_replacement_removes_old_reverse_links() {
    let mut graph = registered(&[1, 2, 3]);
    graph.replace_drivers(3, &[1]).unwrap();
    graph.replace_drivers(3, &[2]).unwrap();
    assert!(graph.readers(1).unwrap().is_empty());
    assert_eq!(graph.readers(2).unwrap(), [3]);
    graph.replace_drivers(3, &[]).unwrap();
    assert!(graph.drivers(3).unwrap().is_empty());
    assert!(graph.readers(2).unwrap().is_empty());
}

#[test]
fn failed_cycle_edit_preserves_existing_graph() {
    let mut graph = registered(&[1, 2, 3]);
    graph.replace_drivers(2, &[1]).unwrap();
    graph.replace_drivers(3, &[2]).unwrap();
    assert_eq!(
        graph.replace_drivers(2, &[3]),
        Err(DependencyError::Cycle { id: 2, driver: 3 })
    );
    assert_eq!(graph.drivers(2).unwrap(), [1]);
    assert_eq!(graph.recompute_order(1).unwrap(), [1, 2, 3]);
    assert_eq!(
        graph.replace_drivers(1, &[1]),
        Err(DependencyError::Cycle { id: 1, driver: 1 })
    );
}

#[test]
fn missing_driver_edit_is_atomic_and_duplicate_registration_is_refused() {
    let mut graph = registered(&[1, 2]);
    graph.replace_drivers(2, &[1]).unwrap();
    assert_eq!(
        graph.replace_drivers(2, &[88]),
        Err(DependencyError::Missing(88))
    );
    assert_eq!(graph.insert(2), Err(DependencyError::Duplicate(2)));
    assert_eq!(graph.drivers(2).unwrap(), [1]);
    assert_eq!(graph.readers(1).unwrap(), [2]);
    assert_eq!(graph.recompute_order(88), Err(DependencyError::Missing(88)));
    assert_eq!(graph.remove(88), Err(DependencyError::Missing(88)));
}

#[test]
fn deletion_names_blockers_and_detaches_unused_node() {
    let mut graph = registered(&[1, 3, 2]);
    graph.replace_drivers(2, &[1]).unwrap();
    graph.replace_drivers(3, &[1]).unwrap();
    assert_eq!(
        graph.remove(1),
        Err(DependencyError::InUse {
            id: 1,
            readers: vec![3, 2]
        })
    );
    graph.remove(3).unwrap();
    assert_eq!(graph.readers(1).unwrap(), [2]);
    graph.remove(2).unwrap();
    graph.remove(1).unwrap();
    assert_eq!(graph.drivers(1), Err(DependencyError::Missing(1)));
}

#[test]
fn cloned_graph_has_independent_edges() {
    let mut original = registered(&[1, 2]);
    original.replace_drivers(2, &[1]).unwrap();
    let mut snapshot = original.clone();
    snapshot.replace_drivers(2, &[]).unwrap();
    assert_eq!(original.recompute_order(1).unwrap(), [1, 2]);
    assert_eq!(snapshot.recompute_order(1).unwrap(), [1]);
    assert!(format!("{original:?}").contains("DependencyGraph"));
}

#[test]
fn deep_imported_chain_does_not_use_recursive_traversal() {
    let ids: Vec<u32> = (0..10_000).collect();
    let mut graph = registered(&ids);
    for id in 1..10_000 {
        graph.replace_drivers(id, &[id - 1]).unwrap();
    }
    assert_eq!(graph.recompute_order(0).unwrap(), ids);
    assert_eq!(
        graph.replace_drivers(0, &[9_999]),
        Err(DependencyError::Cycle {
            id: 0,
            driver: 9_999
        })
    );
}

#[test]
fn errors_name_invalid_values_and_expected_shape() {
    let errors = [
        DependencyError::Missing(88),
        DependencyError::Duplicate(88),
        DependencyError::Cycle { id: 88, driver: 99 },
        DependencyError::InUse {
            id: 88,
            readers: vec![99],
        },
    ];
    for error in errors {
        let diagnostic = error.to_string();
        assert!(diagnostic.contains("88"), "{diagnostic}");
        assert!(diagnostic.contains("expected"), "{diagnostic}");
        assert!(std::error::Error::source(&error).is_none());
    }
}

#[test]
fn all_documents_order_drivers_first_even_when_readers_opened_first() {
    let mut graph = registered(&[3, 2, 1, 4]);
    graph.replace_drivers(3, &[2]).unwrap();
    graph.replace_drivers(2, &[1]).unwrap();
    assert_eq!(graph.recompute_all(), [1, 2, 3, 4]);
    assert!(DependencyGraph::<u32>::default().recompute_all().is_empty());
}
