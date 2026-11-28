// SPDX-License-Identifier: GPL-2.0-only
//! Verify v2 Fillet geometry (variable radius, Smooth G2, multi-set, shared vertices, loops).
use super::*;
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::feature::{
    FeatureEvaluationStatus,
    fillet::FilletDefinition,
    fillet_options::{
        FilletContinuity, FilletEdgeSet, FilletMode, FilletOptions, FilletVariablePoint,
    },
};

#[test]
fn fillet_v2_variable_g2_multiset_and_shared_vertices() {
    let history = owner(cube_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let base = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let solid = base.body(FeatureId(1)).unwrap();
    let edges = solid.edge_reference_keys(&history, document()).unwrap();
    assert_eq!(edges.len(), 12, "cube must have 12 named edges");

    // Pick 3 edges meeting at a shared vertex (0,0,0)
    let mut corner_edges = Vec::new();
    for edge in &edges {
        let segs = solid
            .edge_display_segments(&history, document(), edge, 0.02)
            .unwrap();
        let touches_origin = [segs[0][0], segs[segs.len() - 1][1]]
            .iter()
            .any(|p| p[0].hypot(p[1]).hypot(p[2]) < 1e-3);
        if touches_origin {
            corner_edges.push(edge.clone());
        }
    }
    assert_eq!(corner_edges.len(), 3, "origin must have 3 incident edges");
    let shared = solid
        .shared_fillet_vertices(&history, document(), &corner_edges)
        .unwrap();
    assert_eq!(shared.len(), 1, "3 corner edges must share 1 vertex");
    assert_eq!(
        solid
            .shared_fillet_vertices(&history, document(), &corner_edges[..1])
            .unwrap()
            .len(),
        0,
        "1 edge must report 0 shared vertices"
    );

    // Variable radius + Smooth G2 on a single edge
    let mut var_history = owner(cube_document());
    let var_def = FilletDefinition::new_with_options(
        key("body", &[1]),
        vec![edges[0].clone()],
        DatumLength::Constant(0.2),
        FilletOptions {
            mode: FilletMode::Variable,
            continuity: FilletContinuity::SmoothG2,
            smooth_radius_transition: true,
            variable_points: vec![
                FilletVariablePoint {
                    position: 0.0,
                    radius: DatumLength::Constant(0.2),
                },
                FilletVariablePoint {
                    position: 0.5,
                    radius: DatumLength::Constant(0.1),
                },
            ],
            ..Default::default()
        },
    )
    .unwrap();
    let tok = var_history.begin("VarFillet").unwrap();
    let fid = var_history.create_fillet(&tok, "Fillet1", var_def).unwrap();
    var_history.commit(&tok, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&var_history, document(), certified_resolution());
    let res = var_history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        res.health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy),
        "var fillet health: {:?}",
        res.health()
    );
    let vol = res
        .body(fid)
        .unwrap()
        .volume(&var_history, document())
        .unwrap();
    assert!(
        vol > 0.9 && vol < 1.0,
        "expected filleted volume < 1.0, got {vol}"
    );

    // Multi-edge-set fillet with distinct radii
    let mut set_history = owner(cube_document());
    let set_def = FilletDefinition::new_with_options(
        key("body", &[1]),
        vec![edges[0].clone(), edges[1].clone()],
        DatumLength::Constant(0.15),
        FilletOptions {
            edge_sets: vec![
                FilletEdgeSet {
                    edges: vec![edges[0].clone()],
                    radius: DatumLength::Constant(0.1),
                },
                FilletEdgeSet {
                    edges: vec![edges[1].clone()],
                    radius: DatumLength::Constant(0.2),
                },
            ],
            ..Default::default()
        },
    )
    .unwrap();
    let tok = set_history.begin("MultiSet").unwrap();
    let fid = set_history.create_fillet(&tok, "Fillet1", set_def).unwrap();
    set_history.commit(&tok, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&set_history, document(), certified_resolution());
    let res = set_history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        res.health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy),
        "multi-set health: {:?}",
        res.health()
    );
    let vol = res
        .body(fid)
        .unwrap()
        .volume(&set_history, document())
        .unwrap();
    assert!(
        vol > 0.9 && vol < 1.0,
        "expected multi-set volume < 1.0, got {vol}"
    );
}
