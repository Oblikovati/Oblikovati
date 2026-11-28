// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Surface Stitch via `BRepBuilderAPISewing` (#3918).
use super::*;
use oblikovati_martensite::analyze_stitch_surfaces;
use oblikovati_model::feature::{
    AuthoredFeature, FeatureEvaluationStatus,
    stitch::{StitchDefinition, StitchRecipe},
    thicken_offset::{
        ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean, ThickenDirection,
        ThickenOffsetDefinition, ThickenOffsetRecipe, ThickenOutputMode,
    },
};

fn zero_offset_surface_recipe(body: Vec<u8>, face: Vec<u8>) -> ThickenOffsetRecipe {
    ThickenOffsetRecipe {
        version: 1,
        target_body: body,
        faces: vec![face],
        output: ThickenOutputMode::Surface,
        distance: 0.0,
        direction: ThickenDirection::Positive,
        boolean: ThickenBoolean::NewSolid,
        automatic_face_chain: false,
        automatic_blending: false,
        allow_approximation: false,
        approximation_type: ThickenApproximationType::Mean,
        approximation_tolerance: ThickenApproximationTolerance::Optimized,
        approximation_percent: None,
    }
}

fn find_face_by_max_axis(
    history: &DocumentEditHistory,
    doc: DocumentGuid,
    body: &NamedSolid,
    axis: usize,
) -> Vec<u8> {
    body.face_reference_keys(history, doc)
        .unwrap()
        .into_iter()
        .max_by(|a, b| {
            let ca = body.face_bounds_center(history, doc, a).unwrap()[axis];
            let cb = body.face_bounds_center(history, doc, b).unwrap()[axis];
            ca.total_cmp(&cb)
        })
        .unwrap()
}

#[test]
fn stitch_sews_two_adjacent_zero_offset_surfaces_and_reports_exact_free_edges() {
    // Base block: 4 x 3 x 2 cm. Create two zero-offset surfaces from top (+Z) and side (+Y) faces.
    let mut snapshot = selected_prism_document();
    let doc = document();
    let base_history = owner(snapshot.clone());
    let mut base_eval =
        CertifiedNamedSolidEvaluator::new(&base_history, doc, certified_resolution());
    let base_part = base_history
        .snapshot()
        .evaluate_part_features(&mut base_eval)
        .unwrap();
    let block = base_part.body(FeatureId(1)).unwrap();
    let body_key = block
        .body_reference_key(&base_history, doc)
        .unwrap()
        .to_vec();
    let top_face = find_face_by_max_axis(&base_history, doc, block, 2);
    let side_face = find_face_by_max_axis(&base_history, doc, block, 1);
    assert_ne!(top_face, side_face);

    let def1 = ThickenOffsetDefinition::from_recipe(zero_offset_surface_recipe(
        body_key.clone(),
        top_face,
    ))
    .unwrap();
    let reads1 = def1.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "OffsetSrf1",
            PartFeatureDefinition::ThickenOffset(Box::new(def1)),
            reads1,
        ))
        .unwrap();

    let def2 =
        ThickenOffsetDefinition::from_recipe(zero_offset_surface_recipe(body_key, side_face))
            .unwrap();
    let reads2 = def2.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "OffsetSrf2",
            PartFeatureDefinition::ThickenOffset(Box::new(def2)),
            reads2,
        ))
        .unwrap();

    // Replay the two zero-offset surfaces and extract their body reference keys.
    let srf_history = owner(snapshot.clone());
    let mut srf_eval = CertifiedNamedSolidEvaluator::new(&srf_history, doc, certified_resolution());
    let srf_part = srf_history
        .snapshot()
        .evaluate_part_features(&mut srf_eval)
        .unwrap();
    let srf1 = srf_part.body(FeatureId(2)).unwrap();
    let srf2 = srf_part.body(FeatureId(3)).unwrap();
    assert!(srf1.is_surface());
    assert!(srf2.is_surface());
    let srf1_key = srf1.body_reference_key(&srf_history, doc).unwrap().to_vec();
    let srf2_key = srf2.body_reference_key(&srf_history, doc).unwrap().to_vec();

    // Direct BRepBuilderAPISewing analysis: 1 shared edge sewn, 6 remaining free edges.
    let analysis = analyze_stitch_surfaces(&[srf1, srf2], &srf_history, doc, 0.05).unwrap();
    assert_eq!(
        analysis.stitched_edge_count, 1,
        "expected exactly 1 shared edge sewn between adjacent planar faces"
    );
    assert_eq!(
        analysis.free_edges.len(),
        6,
        "expected 6 remaining free boundary edges on two joined rectangles"
    );
    assert!(!analysis.closed_shell);
    for edge in &analysis.free_edges {
        assert!(
            edge.length_mm > 1.0,
            "expected positive B-rep edge length, got {edge:?}"
        );
    }

    // Append Stitch Surface1 with 0.500 mm tolerance and Maintain as surface = true.
    let stitch_def = StitchDefinition::from_recipe(StitchRecipe {
        version: 1,
        surfaces: vec![srf1_key, srf2_key],
        maintain_as_surface: true,
        maximum_tolerance_mm: 0.5,
        show_edge_conditions: true,
        show_near_tangent: false,
    })
    .unwrap();
    let stitch_reads = stitch_def.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(4),
            "Stitch Surface1",
            PartFeatureDefinition::Stitch(Box::new(stitch_def)),
            stitch_reads,
        ))
        .unwrap();

    // Verify v74 program / v64 source recipe round-trip and tolerance persistence.
    let program = snapshot.work_geometry.part_recipe().unwrap();
    assert_eq!(program.version, 74);
    assert_eq!(program.features[3].definition.version, 64);
    let restored = oblikovati_model::feature::FeatureSequence::from_part_recipe(&program).unwrap();
    let PartFeatureDefinition::Stitch(restored_stitch) = restored.features()[3].definition() else {
        panic!("expected restored Stitch feature");
    };
    assert!((restored_stitch.maximum_tolerance_mm() - 0.5).abs() < 1e-12);
    assert_eq!(restored_stitch.surfaces().len(), 2);

    // Replay the full part with Stitch Surface1 and verify the stitched surface shell.
    let history = owner(snapshot);
    let mut eval = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let part = history
        .snapshot()
        .evaluate_part_features(&mut eval)
        .unwrap();
    assert!(
        part.health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy),
        "expected all features healthy: {:?}",
        part.health()
    );
    let stitched = part.body(FeatureId(4)).unwrap();
    assert!(
        stitched.is_surface(),
        "expected Maintain as surface to produce a surface shell"
    );
    let surface_count = part.ordered_bodies().filter(|b| b.is_surface()).count();
    assert_eq!(
        surface_count, 3,
        "expected 2 input surfaces + 1 stitched surface = 3 surface bodies"
    );
    let stitched_key = stitched.body_reference_key(&history, doc).unwrap();
    let summary = stitched
        .topology_summary(&history, doc, stitched_key)
        .unwrap();
    assert_eq!(summary.faces, 2, "expected 2 faces in sewn shell");
    assert_eq!(
        summary.edges, 7,
        "expected 7 unique edges (1 sewn + 6 free) in sewn shell"
    );
}
