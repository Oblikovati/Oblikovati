// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Ruled Surface modes (Normal, Tangent, Sweep), Angle, Flip, and Edit (#3918).
use super::*;
use oblikovati_model::feature::{
    AuthoredFeature,
    ruled_surface::{
        RuledSurfaceDefinition, RuledSurfaceRecipe, RuledSurfaceType, RuledSurfaceVector,
    },
    thicken_offset::{
        ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean, ThickenDirection,
        ThickenOffsetDefinition, ThickenOffsetRecipe, ThickenOutputMode,
    },
};

fn setup_offset_surface_snapshot() -> (DocumentSnapshot, Vec<u8>, Vec<Vec<u8>>) {
    // Base block: 4 x 3 x 2 cm; top face perimeter = 2 * (4 + 3) = 14 cm.
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
    let block_key = block
        .body_reference_key(&base_history, doc)
        .unwrap()
        .to_vec();
    let top_face = block
        .face_reference_keys(&base_history, doc)
        .unwrap()
        .into_iter()
        .max_by(|a, b| {
            let za = block.face_bounds_center(&base_history, doc, a).unwrap()[2];
            let zb = block.face_bounds_center(&base_history, doc, b).unwrap()[2];
            za.total_cmp(&zb)
        })
        .unwrap();

    let offset_def = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        version: 1,
        target_body: block_key,
        faces: vec![top_face],
        output: ThickenOutputMode::Surface,
        distance: 0.1,
        direction: ThickenDirection::Positive,
        boolean: ThickenBoolean::Join,
        automatic_face_chain: false,
        automatic_blending: false,
        allow_approximation: false,
        approximation_type: ThickenApproximationType::Mean,
        approximation_tolerance: ThickenApproximationTolerance::Optimized,
        approximation_percent: None,
    })
    .unwrap();
    let reads = offset_def.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "OffsetSrf1",
            PartFeatureDefinition::ThickenOffset(Box::new(offset_def)),
            reads,
        ))
        .unwrap();

    let offset_history = owner(snapshot.clone());
    let mut offset_eval =
        CertifiedNamedSolidEvaluator::new(&offset_history, doc, certified_resolution());
    let offset_part = offset_history
        .snapshot()
        .evaluate_part_features(&mut offset_eval)
        .unwrap();
    let srf = offset_part.body(FeatureId(2)).unwrap();
    assert!(srf.is_surface());
    let srf_key = srf
        .body_reference_key(&offset_history, doc)
        .unwrap()
        .to_vec();
    let first_edge = srf
        .subentity_reference_keys(&offset_history, doc)
        .unwrap()
        .into_iter()
        .find(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .is_ok_and(|env| env.kind == 2)
        })
        .unwrap();
    let chained = srf
        .boundary_edge_chain(&offset_history, doc, &first_edge)
        .unwrap();
    assert_eq!(
        chained.len(),
        4,
        "rectangular offset surface chains 4 edges"
    );
    (snapshot, srf_key, chained)
}

fn total_surface_area(history: &DocumentEditHistory, doc: DocumentGuid, body: &NamedSolid) -> f64 {
    body.face_reference_keys(history, doc)
        .unwrap()
        .iter()
        .map(|face| body.face_area(history, doc, face).unwrap())
        .sum()
}

#[test]
fn ruled_surface_normal_tangent_sweep_angle_flip_and_recipe_roundtrip() {
    let (base, srf_key, chained) = setup_offset_surface_snapshot();
    let doc = document();

    // 1. Normal mode with 4 chained edges, distance = 1.0 cm (10 mm): perimeter 14 cm * 1 cm = 14 cm^2.
    let mut snap_normal = base.clone();
    let def_normal = RuledSurfaceDefinition::from_recipe(RuledSurfaceRecipe {
        version: 1,
        target_body: srf_key.clone(),
        edges: chained.clone(),
        ruled_type: RuledSurfaceType::Normal,
        distance: 1.0,
        angle_deg: 0.0,
        flip: false,
        automatic_edge_chain: true,
        vector: None,
    })
    .unwrap();
    let reads = def_normal.reads();
    snap_normal
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Ruled Surface1",
            PartFeatureDefinition::RuledSurface(Box::new(def_normal)),
            reads,
        ))
        .unwrap();
    let recipe = snap_normal.work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 76);
    assert_eq!(recipe.features.last().unwrap().definition.version, 66);
    let restored = oblikovati_model::feature::FeatureSequence::from_part_recipe(&recipe).unwrap();
    assert_eq!(restored.features().len(), 3);

    let history = owner(snap_normal);
    let mut eval = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let part = history
        .snapshot()
        .evaluate_part_features(&mut eval)
        .unwrap();
    let normal_srf = part.body(FeatureId(3)).unwrap();
    assert!(normal_srf.is_surface());
    assert_eq!(
        normal_srf.face_reference_keys(&history, doc).unwrap().len(),
        4
    );
    let area = total_surface_area(&history, doc, normal_srf);
    assert!(
        (area - 14.0).abs() < 1.0e-3,
        "expected 14.0 cm^2, got {area}"
    );

    // 2. Tangent mode with 1 edge, distance = 0.5 cm.
    let mut snap_tan = base.clone();
    let def_tan = RuledSurfaceDefinition::from_recipe(RuledSurfaceRecipe {
        version: 1,
        target_body: srf_key.clone(),
        edges: vec![chained[0].clone()],
        ruled_type: RuledSurfaceType::Tangent,
        distance: 0.5,
        angle_deg: 0.0,
        flip: false,
        automatic_edge_chain: false,
        vector: None,
    })
    .unwrap();
    let reads = def_tan.reads();
    snap_tan
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Ruled Surface1",
            PartFeatureDefinition::RuledSurface(Box::new(def_tan)),
            reads,
        ))
        .unwrap();
    let tan_history = owner(snap_tan);
    let mut tan_eval = CertifiedNamedSolidEvaluator::new(&tan_history, doc, certified_resolution());
    let tan_part = tan_history
        .snapshot()
        .evaluate_part_features(&mut tan_eval)
        .unwrap();
    let tan_srf = tan_part.body(FeatureId(3)).unwrap();
    assert!(tan_srf.is_surface());
    assert_eq!(
        tan_srf
            .face_reference_keys(&tan_history, doc)
            .unwrap()
            .len(),
        1
    );
    let edge_len = tan_part
        .body(FeatureId(2))
        .unwrap()
        .edge_length(&tan_history, doc, &chained[0])
        .unwrap();
    let tan_area = total_surface_area(&tan_history, doc, tan_srf);
    assert!(
        (tan_area - edge_len * 0.5).abs() < 1.0e-3,
        "expected {}, got {tan_area}",
        edge_len * 0.5
    );

    // 3. Sweep mode with Z Axis, distance = 2.0 cm (20 mm), angles 0 / 15 / 180 deg, and flip arrow check.
    for &(angle_deg, flip) in &[(0.0, false), (0.0, true), (15.0, false), (180.0, false)] {
        let mut snap = base.clone();
        let def = RuledSurfaceDefinition::from_recipe(RuledSurfaceRecipe {
            version: 1,
            target_body: srf_key.clone(),
            edges: vec![chained[0].clone()],
            ruled_type: RuledSurfaceType::Sweep,
            distance: 2.0,
            angle_deg,
            flip,
            automatic_edge_chain: false,
            vector: Some(RuledSurfaceVector::OriginZ),
        })
        .unwrap();
        let resolved = def.resolve_with_direction(Some([0.0, 0.0, 1.0]), None);
        let reads = def.reads();
        snap.work_geometry
            .append(AuthoredFeature::new(
                FeatureId(3),
                "Ruled Surface1",
                PartFeatureDefinition::RuledSurface(Box::new(def)),
                reads,
            ))
            .unwrap();
        let h = owner(snap);
        let mut ev = CertifiedNamedSolidEvaluator::new(&h, doc, certified_resolution());
        let p = h.snapshot().evaluate_part_features(&mut ev).unwrap();
        let swept = p.body(FeatureId(3)).unwrap();
        assert!(swept.is_surface());
        let swept_area = total_surface_area(&h, doc, swept);
        assert!(
            (swept_area - edge_len * 2.0).abs() < 1.0e-3,
            "angle={angle_deg}, flip={flip}: expected {}, got {swept_area}",
            edge_len * 2.0
        );
        let arrow = p
            .body(FeatureId(2))
            .unwrap()
            .ruled_surface_preview_arrow(&h, doc, &resolved)
            .unwrap()
            .unwrap();
        if angle_deg == 0.0 && !flip {
            assert!(arrow.direction[2] > 0.99);
        } else if (angle_deg == 0.0 && flip) || (angle_deg == 180.0 && !flip) {
            assert!(arrow.direction[2] < -0.99);
        }
    }
}
