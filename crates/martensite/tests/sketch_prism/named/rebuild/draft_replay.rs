// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Part Modify Draft (FixedEdge, FixedPlane, PartingTool, and 2.5°/5° edits).
use super::*;
use oblikovati_model::feature::{
    draft::{DraftAngleMode, DraftDefinition, DraftMode, DraftPartingTool, DraftRecipe},
    hole::HoleOriginPlane,
};

fn evaluate_draft_volume(
    mode: DraftMode,
    parting_tool: Option<DraftPartingTool>,
    angle_deg: f64,
    angle_mode: DraftAngleMode,
) -> f64 {
    let mut history = owner(cube_document());
    let recipe = DraftRecipe {
        version: 1,
        body: key("body", &[1]),
        mode,
        pull_direction: key("end", &[1]),
        parting_tool,
        faces: vec![key("side", &[1, 101])],
        angle_deg,
        angle_mode,
        direction_flipped: false,
        automatic_blending: true,
    };
    let definition = DraftDefinition::from_recipe(recipe).unwrap();
    let token = history.begin("FaceDraft").unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_draft("FaceDraft1", definition, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        result
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        result.health()
    );
    let body = result.body(id).expect("expected Draft solid body");
    let summary = body
        .topology_summary(&history, document(), &key("body", &[1]))
        .unwrap();
    assert!(summary.solid, "expected valid solid: {summary:?}");
    assert_eq!(summary.faces, 6, "expected 6 faces on drafted cube");
    body.volume(&history, document()).unwrap()
}

#[test]
fn drafted_block_faces_match_analytic_wedge_volumes_for_2_5_and_5_deg() {
    // Unit cube 1 x 1 x 1 cm; drafting one side face by angle `a` about z = 1 (or z = 0)
    // changes volume by the exact triangular prism wedge: 0.5 * 1 * (1 * tan(a)) * 1 = 0.5 * tan(a).
    for angle_deg in [2.5_f64, 5.0_f64] {
        let wedge = 0.5 * angle_deg.to_radians().tan();
        let v_edge = evaluate_draft_volume(
            DraftMode::FixedEdge,
            None,
            angle_deg,
            DraftAngleMode::Absolute,
        );
        assert!(
            ((v_edge - 1.0).abs() - wedge).abs() < 1e-6,
            "FixedEdge {angle_deg}°: volume {v_edge}, expected |v-1| == {wedge}"
        );

        let v_plane = evaluate_draft_volume(
            DraftMode::FixedPlane,
            None,
            angle_deg,
            DraftAngleMode::Absolute,
        );
        assert!(
            ((v_plane - 1.0).abs() - wedge).abs() < 1e-6,
            "FixedPlane {angle_deg}°: volume {v_plane}, expected |v-1| == {wedge}"
        );

        let v_parting = evaluate_draft_volume(
            DraftMode::PartingTool,
            Some(DraftPartingTool::OriginPlane(HoleOriginPlane::Xz)),
            angle_deg,
            DraftAngleMode::Relative,
        );
        assert!(
            ((v_parting - 1.0).abs() - wedge).abs() < 1e-6,
            "PartingTool XZ {angle_deg}°: volume {v_parting}, expected |v-1| == {wedge}"
        );
    }

    // Verify editing from 2.5° to 5° strictly changes volume by the wedge difference.
    let v_2_5 = evaluate_draft_volume(DraftMode::FixedEdge, None, 2.5, DraftAngleMode::Absolute);
    let v_5 = evaluate_draft_volume(DraftMode::FixedEdge, None, 5.0, DraftAngleMode::Absolute);
    assert!(
        (v_5 - v_2_5).abs() > 0.02,
        "expected distinct volumes for 2.5° ({v_2_5}) vs 5° ({v_5})"
    );
}
