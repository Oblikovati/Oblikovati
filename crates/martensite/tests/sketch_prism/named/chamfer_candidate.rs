// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn named_edge_chamfer_preserves_original_and_resolves_result_names() {
    let history = owner(cube_document());
    let body = certified(&history);
    let edges = body.subentity_reference_keys(&history, document()).unwrap();
    let edge = edges
        .iter()
        .find(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .unwrap()
                .kind
                == 2
        })
        .unwrap();
    let candidate = body
        .chamfer_candidate(&history, document(), std::slice::from_ref(edge), 0.2)
        .unwrap();
    assert!((candidate.volume().unwrap() - 0.98).abs() < 1e-8);
    let names = candidate
        .name_topology(&history, document(), FeatureId(2))
        .unwrap();
    assert!(names.deleted_keys().contains(edge));
    for key in names.live_keys() {
        let kind = oblikovati_model::identity::ReferenceEnvelope::decode(key)
            .unwrap()
            .kind;
        assert!(names.contains(&history, document(), key, kind).unwrap());
    }
    let original = body
        .physical_properties(
            &history,
            document(),
            body.body_reference_key(&history, document()).unwrap(),
        )
        .unwrap();
    assert!((original.geometry.volume - 1.).abs() < 1e-8);
}

#[test]
fn named_edge_chamfer_rejects_invalid_inputs() {
    let history = owner(cube_document());
    let body = certified(&history);
    let edge = body
        .subentity_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .unwrap()
                .kind
                == 2
        })
        .unwrap();
    for (edges, distance) in [
        (vec![], 0.1),
        (vec![edge.clone(), edge.clone()], 0.1),
        (vec![edge.clone()], 0.),
        (vec![edge.clone()], f64::NAN),
        (vec![key("start", &[1])], 0.1),
        (vec![b"foreign".to_vec()], 0.1),
    ] {
        assert!(
            body.chamfer_candidate(&history, document(), &edges, distance)
                .is_err()
        );
    }
}

#[test]
fn chamfer_then_fillet_retains_deleted_edge_names() {
    let history = owner(cube_document());
    let body = certified(&history);
    let edges = opposite_cube_edges(&body, &history);
    let first = body
        .chamfer_candidate(&history, document(), &edges[..1], 0.1)
        .unwrap()
        .name_topology(&history, document(), FeatureId(2))
        .unwrap();
    assert!(
        first
            .chamfer_candidate(&history, document(), &edges[..1], 0.1)
            .is_err()
    );
    let second = first
        .fillet_candidate(&history, document(), &edges[1..], 0.1)
        .unwrap()
        .name_topology(&history, document(), FeatureId(3))
        .unwrap();
    assert!(
        edges
            .iter()
            .all(|edge| second.deleted_keys().contains(edge))
    );
}

#[test]
fn fillet_then_chamfer_keeps_prior_names_and_changes_volume() {
    let history = owner(cube_document());
    let body = certified(&history);
    let edges = opposite_cube_edges(&body, &history);
    let first = body
        .fillet_candidate(&history, document(), &edges[..1], 0.1)
        .unwrap()
        .name_topology(&history, document(), FeatureId(2))
        .unwrap();
    let second = first
        .chamfer_candidate(&history, document(), &edges[1..], 0.1)
        .unwrap();
    let expected = 1. - (0.01 - std::f64::consts::PI * 0.0025) - 0.005;
    assert!((second.volume().unwrap() - expected).abs() < 1e-8);
    let named = second
        .name_topology(&history, document(), FeatureId(3))
        .unwrap();
    assert!(edges.iter().all(|edge| named.deleted_keys().contains(edge)));
}

fn opposite_cube_edges(body: &NamedPrismSnapshot, history: &DocumentEditHistory) -> Vec<Vec<u8>> {
    use oblikovati_martensite::{PrismHistorySource, PrismSubentityRole};
    let edges: Vec<_> = body
        .subentity_reference_keys(history, document())
        .unwrap()
        .into_iter()
        .filter(|key| {
            let provenance = body.subentity_provenance(history, document(), key).unwrap();
            matches!(
                provenance.role,
                PrismSubentityRole::StartEdge | PrismSubentityRole::EndEdge
            ) && provenance.source == PrismHistorySource::Curve(101)
        })
        .collect();
    assert_eq!(edges.len(), 2);
    edges
}

#[test]
fn chamfer_topology_names_are_stable_across_distance_edits() {
    let history = owner(cube_document());
    let body = certified(&history);
    let edge = opposite_cube_edges(&body, &history).remove(0);
    let names = |distance| {
        body.chamfer_candidate(&history, document(), std::slice::from_ref(&edge), distance)
            .unwrap()
            .name_topology(&history, document(), FeatureId(2))
            .unwrap()
    };
    let first = names(0.1);
    let second = names(0.2);
    assert_eq!(first.live_keys(), second.live_keys());
    assert_eq!(first.deleted_keys(), second.deleted_keys());
    assert!(first.deleted_keys().contains(&edge));
}

#[test]
fn chamfer_accepts_two_named_edges_and_preserves_their_deletion_history() {
    let history = owner(cube_document());
    let body = certified(&history);
    let edges = opposite_cube_edges(&body, &history);
    let candidate = body
        .chamfer_candidate(&history, document(), &edges, 0.1)
        .unwrap();
    assert!((candidate.volume().unwrap() - 0.99).abs() < 1e-8);
    let named = candidate
        .name_topology(&history, document(), FeatureId(2))
        .unwrap();
    assert!(edges.iter().all(|edge| named.deleted_keys().contains(edge)));
}
