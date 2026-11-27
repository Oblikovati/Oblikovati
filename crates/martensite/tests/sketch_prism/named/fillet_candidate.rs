// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn exact_segment_pick_reports_named_edges_in_depth_order() {
    use oblikovati_martensite::NamedSolid;
    use oblikovati_model::identity::SemanticReferenceName;
    let history = owner(cube_document());
    let body = NamedSolid::Prism(Box::new(certified(&history)));
    let hits = body
        .edge_hits_on_segment(&history, document(), [[0.5, -1., -1.], [0.5, 2., 2.]])
        .unwrap();
    let expected = SemanticReferenceName::new(
        document(),
        1,
        2,
        "sketch-prism",
        "start-edge-curve",
        &[1, 101],
    )
    .unwrap()
    .encode()
    .unwrap();
    let hit = hits.iter().find(|hit| hit.key == expected).unwrap();
    assert!(hit.distance < 1e-8, "distance {}", hit.distance);
    assert!(
        (hit.reach - 2f64.sqrt()).abs() < 1e-8,
        "reach {}",
        hit.reach
    );
    let segments = body
        .edge_display_segments(&history, document(), &expected, 0.01)
        .unwrap();
    assert!(!segments.is_empty());
    assert!(
        segments
            .iter()
            .flatten()
            .flatten()
            .all(|value| value.is_finite())
    );
    assert!(
        body.edge_hits_on_segment(&history, document(), [[0.; 3], [0.; 3]])
            .is_err()
    );
    assert!(
        body.edge_hits_on_ray(&history, document(), [[20., 20., 3.], [0., 0., -1.]], 0.01)
            .unwrap()
            .is_empty()
    );
    assert!(
        body.edge_hits_on_ray(
            &history,
            document(),
            [[0.5, -0.005, 3.], [0., 0., -1.]],
            0.01
        )
        .unwrap()
        .iter()
        .any(|hit| hit.distance <= 0.01)
    );
}

#[test]
#[ignore = "manual release-mode latency measurement"]
fn measure_named_edge_pick_latency() {
    let history = owner(cube_document());
    let solid = oblikovati_martensite::NamedSolid::Prism(Box::new(certified(&history)));
    let segment = [[0.5, -1., -1.], [0.5, 2., 2.]];
    for _ in 0..3 {
        std::hint::black_box(
            solid
                .edge_hits_on_segment(&history, document(), segment)
                .unwrap(),
        );
    }
    let mut samples = Vec::new();
    for _ in 0..30 {
        let start = std::time::Instant::now();
        let hits = solid
            .edge_hits_on_segment(&history, document(), segment)
            .unwrap();
        std::hint::black_box(hits);
        samples.push(start.elapsed().as_secs_f64() * 1000.);
    }
    samples.sort_by(f64::total_cmp);
    println!(
        "named-edge pick milliseconds: median {:.3}, p95 {:.3}, max {:.3}",
        samples[15], samples[28], samples[29]
    );
}

#[test]
fn named_edge_builds_detached_fillet_and_retains_kernel_history() {
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
    let original = body
        .physical_properties(
            &history,
            document(),
            body.body_reference_key(&history, document()).unwrap(),
        )
        .unwrap()
        .geometry
        .volume;
    let candidate = body
        .fillet_candidate(&history, document(), std::slice::from_ref(&edge), 0.2)
        .unwrap();
    let expected = 1. - (0.04 - std::f64::consts::PI * 0.01);
    assert!((candidate.volume().unwrap() - expected).abs() < 1e-8);
    assert!((original - 1.).abs() < 1e-8);
    let selected = candidate
        .history()
        .iter()
        .find(|record| record.name() == edge)
        .unwrap();
    assert!(selected.generated_count() > 0);
    assert_eq!(selected.modified_count(), 0);
    assert!(selected.is_deleted());
    let after = body
        .physical_properties(
            &history,
            document(),
            body.body_reference_key(&history, document()).unwrap(),
        )
        .unwrap()
        .geometry
        .volume;
    assert!((after - original).abs() < 1e-12);
}

#[test]
fn fillet_candidate_rejects_invalid_edge_and_radius_inputs() {
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
    let face = key("start", &[1]);
    for (edges, radius) in [
        (vec![], 0.2),
        (vec![edge.clone(), edge.clone()], 0.2),
        (vec![edge.clone()], 0.),
        (vec![edge.clone()], f64::NAN),
        (vec![face], 0.2),
        (vec![b"foreign".to_vec()], 0.2),
    ] {
        assert!(
            body.fillet_candidate(&history, document(), &edges, radius)
                .is_err()
        );
    }
}

#[test]
fn fillet_candidate_accepts_two_separate_named_edges() {
    use oblikovati_martensite::{PrismHistorySource, PrismSubentityRole};
    let history = owner(cube_document());
    let body = certified(&history);
    let edges: Vec<_> = body
        .subentity_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .filter(|key| {
            let provenance = body
                .subentity_provenance(&history, document(), key)
                .unwrap();
            matches!(
                provenance.role,
                PrismSubentityRole::StartEdge | PrismSubentityRole::EndEdge
            ) && provenance.source == PrismHistorySource::Curve(101)
        })
        .collect();
    assert_eq!(edges.len(), 2);
    let candidate = body
        .fillet_candidate(&history, document(), &edges, 0.1)
        .unwrap();
    let expected = 1. - 2. * (0.01 - std::f64::consts::PI * 0.0025);
    assert!((candidate.volume().unwrap() - expected).abs() < 1e-8);
    let names = candidate
        .name_topology(&history, document(), FeatureId(2))
        .unwrap();
    for edge in edges {
        assert!(names.deleted_keys().contains(&edge));
    }
}

#[test]
fn named_fillet_result_accepts_another_edge_and_keeps_deleted_history() {
    use oblikovati_martensite::{PrismHistorySource, PrismSubentityRole};
    let history = owner(cube_document());
    let body = certified(&history);
    let edges: Vec<_> = body
        .subentity_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .filter(|key| {
            let provenance = body
                .subentity_provenance(&history, document(), key)
                .unwrap();
            matches!(
                provenance.role,
                PrismSubentityRole::StartEdge | PrismSubentityRole::EndEdge
            ) && provenance.source == PrismHistorySource::Curve(101)
        })
        .collect();
    assert_eq!(edges.len(), 2);
    let first = body
        .fillet_candidate(&history, document(), &edges[..1], 0.1)
        .unwrap()
        .name_topology(&history, document(), FeatureId(2))
        .unwrap();
    let face = key("end", &[1]);
    assert_eq!(
        first.face_plane(&history, document(), &face).unwrap(),
        body.face_plane(&history, document(), &face).unwrap()
    );
    assert!(first.deleted_keys().contains(&edges[0]));
    assert!(
        first
            .fillet_candidate(&history, document(), &edges[..1], 0.1)
            .is_err()
    );
    let second = first
        .fillet_candidate(&history, document(), &edges[1..], 0.1)
        .unwrap();
    let expected = 1. - 2. * (0.01 - std::f64::consts::PI * 0.0025);
    assert!((second.volume().unwrap() - expected).abs() < 1e-8);
    let names = second
        .name_topology(&history, document(), FeatureId(3))
        .unwrap();
    let mesh = names
        .display_triangles(oblikovati_martensite::PrismMeshOptions {
            chord_error: 0.01,
            angular_error: 0.2,
        })
        .unwrap();
    assert!(!mesh.is_empty());
    assert!(
        mesh.iter()
            .flatten()
            .flatten()
            .all(|coordinate| coordinate.is_finite())
    );
    let signed_volume: f64 = mesh
        .iter()
        .map(|[a, b, c]| {
            (a[0] * (b[1] * c[2] - b[2] * c[1])
                + a[1] * (b[2] * c[0] - b[0] * c[2])
                + a[2] * (b[0] * c[1] - b[1] * c[0]))
                / 6.
        })
        .sum();
    assert!(
        (signed_volume - expected).abs() < 0.002,
        "mesh volume {signed_volume}, expected {expected}"
    );
    assert!(edges.iter().all(|edge| names.deleted_keys().contains(edge)));
    for body_key in first.live_keys().iter().filter(|key| {
        oblikovati_model::identity::ReferenceEnvelope::decode(key)
            .unwrap()
            .kind
            == 4
    }) {
        assert!(names.contains(&history, document(), body_key, 4).unwrap());
    }
    let solid = oblikovati_martensite::NamedSolid::Derived(Box::new(names));
    let hits = solid
        .edge_hits_on_segment(&history, document(), [[0.5, -1., -1.], [0.5, 2., 2.]])
        .unwrap();
    assert!(!hits.is_empty());
    assert!(
        hits.iter()
            .all(|hit| !edges.contains(&hit.key) && hit.distance.is_finite())
    );
}
