// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Thicken NewSolid/Join/Cut failure and OffsetSrf surface shells (#3918).
use super::*;
use oblikovati_model::feature::{
    AuthoredFeature, FeatureEvaluationStatus,
    thicken_offset::{
        ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean, ThickenDirection,
        ThickenOffsetDefinition, ThickenOffsetRecipe, ThickenOutputMode,
    },
};

fn base_recipe(body: Vec<u8>, faces: Vec<Vec<u8>>) -> ThickenOffsetRecipe {
    ThickenOffsetRecipe {
        version: 1,
        target_body: body,
        faces,
        output: ThickenOutputMode::Solid,
        distance: 0.1,
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

fn find_top_planar_face(
    history: &DocumentEditHistory,
    doc: DocumentGuid,
    body: &NamedSolid,
) -> Vec<u8> {
    body.face_reference_keys(history, doc)
        .unwrap()
        .into_iter()
        .max_by(|a, b| {
            let za = body.face_bounds_center(history, doc, a).unwrap()[2];
            let zb = body.face_bounds_center(history, doc, b).unwrap()[2];
            za.total_cmp(&zb)
        })
        .unwrap()
}

#[test]
fn thicken_new_solid_join_and_unchanged_cut_match_exact_analytic_volumes() {
    // Base block: 4 x 3 x 2 cm => volume 24 cm^3, top face area = 12 cm^2.
    let base = selected_prism_document();
    let doc = document();
    let base_history = owner(base.clone());
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
    let top_face = find_top_planar_face(&base_history, doc, block);

    // 1. New Solid: +0.5 cm (5 mm) outward on 12 cm^2 face => 6 cm^3 new body, total 2 solid bodies.
    let mut new_solid_snap = base.clone();
    let def_new = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        distance: 0.5,
        boolean: ThickenBoolean::NewSolid,
        ..base_recipe(body_key.clone(), vec![top_face.clone()])
    })
    .unwrap();
    let reads = def_new.reads();
    new_solid_snap
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Thicken1",
            PartFeatureDefinition::ThickenOffset(Box::new(def_new)),
            reads,
        ))
        .unwrap();
    let history = owner(new_solid_snap);
    let mut eval = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let part = history
        .snapshot()
        .evaluate_part_features(&mut eval)
        .unwrap();
    assert_eq!(part.ordered_bodies().count(), 2);
    let slab = part.body(FeatureId(2)).unwrap();
    assert!(!slab.is_surface());
    let slab_vol = slab.volume(&history, doc).unwrap();
    assert!(
        (slab_vol - 6.0).abs() < 1.0e-5,
        "expected 6.0 cm^3, got {slab_vol}"
    );

    // 2. Join: +0.5 cm outward on 4x3x2 cm block => single 4x3x2.5 cm solid (30 cm^3).
    let mut join_snap = base.clone();
    let def_join = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        distance: 0.5,
        boolean: ThickenBoolean::Join,
        ..base_recipe(body_key.clone(), vec![top_face.clone()])
    })
    .unwrap();
    let reads = def_join.reads();
    join_snap
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Thicken1",
            PartFeatureDefinition::ThickenOffset(Box::new(def_join)),
            reads,
        ))
        .unwrap();
    let history = owner(join_snap);
    let mut eval = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let part = history
        .snapshot()
        .evaluate_part_features(&mut eval)
        .unwrap();
    assert_eq!(part.ordered_bodies().count(), 1);
    let joined_vol = part
        .body(FeatureId(2))
        .unwrap()
        .volume(&history, doc)
        .unwrap();
    assert!(
        (joined_vol - 30.0).abs() < 1.0e-5,
        "expected 30.0 cm^3, got {joined_vol}"
    );

    // 3. Cut outward (+0.5 cm): zero interior overlap => authentic UnchangedCut failure!
    let mut cut_snap = base;
    let def_cut = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        distance: 0.5,
        boolean: ThickenBoolean::Cut,
        ..base_recipe(body_key, vec![top_face])
    })
    .unwrap();
    let reads = def_cut.reads();
    cut_snap
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Thicken1",
            PartFeatureDefinition::ThickenOffset(Box::new(def_cut)),
            reads,
        ))
        .unwrap();
    let history = owner(cut_snap);
    let mut eval = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let part = history
        .snapshot()
        .evaluate_part_features(&mut eval)
        .unwrap();
    assert!(
        matches!(
            part.health().last().unwrap().status,
            FeatureEvaluationStatus::Failed(_)
        ),
        "expected outward Cut to fail authentically, got {:?}",
        part.health()
    );
}

#[test]
fn offset_surface_supports_zero_and_positive_offsets_as_surface_bodies() {
    let base = selected_prism_document();
    let doc = document();
    let base_history = owner(base.clone());
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
    let top_face = find_top_planar_face(&base_history, doc, block);

    for (distance, expected_z) in [(0.0, 2.0), (0.1, 2.1)] {
        let mut snap = base.clone();
        let def = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
            output: ThickenOutputMode::Surface,
            distance,
            ..base_recipe(body_key.clone(), vec![top_face.clone()])
        })
        .unwrap();
        let reads = def.reads();
        snap.work_geometry
            .append(AuthoredFeature::new(
                FeatureId(2),
                "OffsetSrf1",
                PartFeatureDefinition::ThickenOffset(Box::new(def)),
                reads,
            ))
            .unwrap();
        let history = owner(snap);
        let mut eval = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
        let part = history
            .snapshot()
            .evaluate_part_features(&mut eval)
            .unwrap();
        assert_eq!(part.ordered_bodies().count(), 2);
        let surface = part.body(FeatureId(2)).unwrap();
        assert!(
            surface.is_surface(),
            "expected OffsetSrf1 to be a surface body"
        );
        let face_key = surface
            .face_reference_keys(&history, doc)
            .unwrap()
            .remove(0);
        let center = surface
            .face_bounds_center(&history, doc, &face_key)
            .unwrap();
        assert!(
            (center[2] - expected_z).abs() < 1.0e-5,
            "distance {distance}: expected z={expected_z}, got {}",
            center[2]
        );
    }
}

#[test]
fn thicken_rounded_cylinder_curved_chain_automatic_blending() {
    use oblikovati_model::{
        document_sketch::DocumentSketchId,
        feature::{
            fillet::{FilletDefinition, FilletRecipe},
            sketch_prism::{SketchDirectedCurve, SketchPrismSelection},
            work_plane::WorkPlaneOrigin,
        },
        sketch::{Sketch2D, SketchEntityId},
    };
    let mut snap = selected_prism_document();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(201), [8.0, 0.0]).unwrap();
    geometry
        .add_circle(SketchEntityId(202), SketchEntityId(201), 1.0)
        .unwrap();
    snap.sketches
        .add_on_origin_plane(
            DocumentSketchId(2),
            "Sketch2",
            WorkPlaneOrigin::XY,
            geometry,
        )
        .unwrap();
    let prism_def = SketchPrismDefinition::new(
        DocumentSketchId(2),
        SketchPrismSelection {
            outer: vec![SketchDirectedCurve {
                curve: SketchEntityId(202),
                reversed: false,
            }],
            holes: vec![],
        },
        [0.0, 0.0, 2.0].map(DatumLength::Constant),
    )
    .unwrap();
    let reads = prism_def.reads(&snap).unwrap();
    snap.work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Extrusion2",
            PartFeatureDefinition::SketchPrism(Box::new(prism_def)),
            reads,
        ))
        .unwrap();
    solve_selected_sketches(&mut snap);
    let doc = document();
    let h1 = owner(snap.clone());
    let mut e1 = CertifiedNamedSolidEvaluator::new(&h1, doc, certified_resolution());
    let p1 = h1.snapshot().evaluate_part_features(&mut e1).unwrap();
    let cyl = p1.body(FeatureId(2)).unwrap();
    let body_key = cyl.body_reference_key(&h1, doc).unwrap().to_vec();
    let top_edge = cyl
        .subentity_reference_keys(&h1, doc)
        .unwrap()
        .into_iter()
        .find(|k| {
            oblikovati_model::identity::ReferenceEnvelope::decode(k).is_ok_and(|e| e.kind == 2)
                && cyl.subentity_point(&h1, doc, k).is_ok_and(|pt| pt[2] > 1.5)
        })
        .unwrap();
    let fillet_def = FilletDefinition::from_recipe(&FilletRecipe {
        version: 1,
        body: body_key.clone(),
        edges: vec![top_edge],
        radius: DatumLength::Constant(0.2),
        options: Default::default(),
    })
    .unwrap();
    let reads = fillet_def.reads();
    snap.work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Fillet1",
            PartFeatureDefinition::Fillet(Box::new(fillet_def)),
            reads,
        ))
        .unwrap();
    let h2 = owner(snap.clone());
    let mut e2 = CertifiedNamedSolidEvaluator::new(&h2, doc, certified_resolution());
    let p2 = h2.snapshot().evaluate_part_features(&mut e2).unwrap();
    let filleted = p2.body(FeatureId(3)).unwrap();
    let cyl_face = filleted
        .face_reference_keys(&h2, doc)
        .unwrap()
        .into_iter()
        .find(|k| {
            filleted
                .face_topology(&h2, doc, k)
                .is_ok_and(|t| t.surface == PrismSurfaceKind::Cylinder)
        })
        .unwrap();
    let chain = filleted.tangent_face_chain(&h2, doc, &cyl_face).unwrap();
    assert_eq!(chain.len(), 2, "expected Cylinder + Torus chain");

    let thicken_def = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        distance: 0.1,
        direction: ThickenDirection::Positive,
        boolean: ThickenBoolean::Join,
        automatic_face_chain: true,
        automatic_blending: true,
        allow_approximation: true,
        ..base_recipe(body_key, chain)
    })
    .unwrap();
    let reads = thicken_def.reads();
    snap.work_geometry
        .append(AuthoredFeature::new(
            FeatureId(4),
            "Thicken1",
            PartFeatureDefinition::ThickenOffset(Box::new(thicken_def)),
            reads,
        ))
        .unwrap();
    let h3 = owner(snap);
    let mut e3 = CertifiedNamedSolidEvaluator::new(&h3, doc, certified_resolution());
    let p3 = h3.snapshot().evaluate_part_features(&mut e3).unwrap();
    assert!(
        p3.health()
            .iter()
            .all(|r| r.status == FeatureEvaluationStatus::Healthy),
        "expected Healthy curved thicken, got {:?}",
        p3.health()
    );
    let before_vol = filleted.volume(&h2, doc).unwrap();
    let thickened = p3.body(FeatureId(4)).unwrap();
    let after_vol = thickened.volume(&h3, doc).unwrap();
    assert!(
        after_vol > before_vol,
        "expected volume increase: before={before_vol}, after={after_vol}"
    );
    let triangles = oblikovati_martensite::BodyPresentationSource::body_face_triangles(
        thickened,
        oblikovati_martensite::PrismMeshOptions {
            chord_error: 1.0e-3,
            angular_error: 0.1,
        },
    )
    .expect("expected clean display meshing with zero defects");
    assert!(!triangles.is_empty());
}
