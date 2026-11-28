// SPDX-License-Identifier: GPL-2.0-only
//! Verify v2 Chamfer geometry (distance-angle, two-distances, reversed, and exact partial-edge cut volume).
use super::*;
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::feature::{
    FeatureEvaluationStatus,
    chamfer::ChamferDefinition,
    chamfer_options::{ChamferDrivenDimension, ChamferMode, ChamferOptions, ChamferPartialSpec},
};

#[test]
fn chamfer_v2_modes_and_partial_edge_exact_volume() {
    let history = owner(cube_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let base = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let solid = base.body(FeatureId(1)).unwrap();
    let base_vol = solid.volume(&history, document()).unwrap();
    let edges = solid.edge_reference_keys(&history, document()).unwrap();
    assert_eq!(edges.len(), 12);
    let (_p0, _p1, edge_len) = solid.edge_span(&history, document(), &edges[0]).unwrap();
    assert!(
        edge_len > 0.5,
        "expected positive edge length, got {edge_len}"
    );

    // 1) Distance & Angle (0.2 cm at 45 deg -> removes 0.5 * 0.2 * 0.2 * edge_len)
    let da_def = ChamferDefinition::with_options(
        key("body", &[1]),
        vec![edges[0].clone()],
        DatumLength::Constant(0.2),
        ChamferOptions {
            mode: ChamferMode::DistanceAndAngle,
            angle_deg: 45.0,
            ..Default::default()
        },
    )
    .unwrap();
    let mut da_hist = owner(cube_document());
    let tok = da_hist.begin("ChamferDA").unwrap();
    let da_id = da_hist
        .edit(&tok, |s, ids| s.create_chamfer("Chamfer1", da_def, ids))
        .unwrap();
    da_hist.commit(&tok, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&da_hist, document(), certified_resolution());
    let da_res = da_hist
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        da_res
            .health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy)
    );
    let da_vol = da_res
        .body(da_id)
        .unwrap()
        .volume(&da_hist, document())
        .unwrap();
    let expected_da = base_vol - 0.5 * 0.2 * 0.2 * edge_len;
    assert!(
        (da_vol - expected_da).abs() < 1e-3,
        "DA volume {da_vol} vs expected {expected_da}"
    );

    // 2) Two Distances (0.1 cm x 0.2 cm -> removes 0.5 * 0.1 * 0.2 * edge_len)
    let td_def = ChamferDefinition::with_options(
        key("body", &[1]),
        vec![edges[0].clone()],
        DatumLength::Constant(0.1),
        ChamferOptions {
            mode: ChamferMode::TwoDistances,
            distance2: Some(DatumLength::Constant(0.2)),
            ..Default::default()
        },
    )
    .unwrap();
    let mut td_hist = owner(cube_document());
    let tok = td_hist.begin("Chamfer2D").unwrap();
    let td_id = td_hist
        .edit(&tok, |s, ids| s.create_chamfer("Chamfer1", td_def, ids))
        .unwrap();
    td_hist.commit(&tok, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&td_hist, document(), certified_resolution());
    let td_res = td_hist
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        td_res
            .health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy)
    );
    let td_vol = td_res
        .body(td_id)
        .unwrap()
        .volume(&td_hist, document())
        .unwrap();
    let expected_td = base_vol - 0.5 * 0.1 * 0.2 * edge_len;
    assert!(
        (td_vol - expected_td).abs() < 1e-3,
        "2D volume {td_vol} vs expected {expected_td}"
    );

    // 3) Partial chamfer (0.1 cm equal distance over 60% of edge length, To Start = 0)
    let partial_len = edge_len * 0.6;
    let to_end = edge_len - partial_len;
    let partial_def = ChamferDefinition::with_options(
        key("body", &[1]),
        vec![edges[0].clone()],
        DatumLength::Constant(0.1),
        ChamferOptions {
            mode: ChamferMode::EqualDistance,
            partial: Some(ChamferPartialSpec {
                edge_length_cm: edge_len,
                to_start_cm: 0.0,
                chamfer_length_cm: partial_len,
                to_end_cm: to_end,
                driven: ChamferDrivenDimension::Chamfer,
            }),
            ..Default::default()
        },
    )
    .unwrap();
    let mut p_hist = owner(cube_document());
    let tok = p_hist.begin("ChamferPartial").unwrap();
    let p_id = p_hist
        .edit(&tok, |s, ids| {
            s.create_chamfer("Chamfer1", partial_def, ids)
        })
        .unwrap();
    p_hist.commit(&tok, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&p_hist, document(), certified_resolution());
    let p_res = p_hist
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        p_res
            .health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy)
    );
    let p_vol = p_res
        .body(p_id)
        .unwrap()
        .volume(&p_hist, document())
        .unwrap();
    let expected_p = base_vol - 0.5 * 0.1 * 0.1 * partial_len;
    assert!(
        (p_vol - expected_p).abs() < 1e-3,
        "partial volume {p_vol} vs expected {expected_p}"
    );
}
