// SPDX-License-Identifier: GPL-2.0-only
//! Martensite unit and ordered replay tests for Part Surface Extend (#3918).
use super::*;
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        AuthoredFeature, FeatureEvaluationStatus,
        extend_surface::{
            ExtendSurfaceDefinition, ExtendSurfaceExtent, ExtendSurfaceMethod,
            ExtendSurfaceOriginPlane, ExtendSurfaceRecipe, ExtendSurfaceTermination,
        },
        sketch_prism::{SketchDirectedCurve, SketchPrismSelection},
        thicken_offset::{
            ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean,
            ThickenDirection, ThickenOffsetDefinition, ThickenOffsetRecipe, ThickenOutputMode,
        },
        work_plane::WorkPlaneOrigin,
    },
    sketch::{Sketch2D, SketchEntityId},
};

fn block_with_offset_surface() -> (DocumentSnapshot, Vec<u8>, Vec<Vec<u8>>) {
    let mut snapshot = selected_prism_document();
    snapshot.work_geometry = Default::default();
    snapshot.sketches = Default::default();
    let mut geometry = Sketch2D::default();
    // 40 mm x 25 mm rectangle matching body-properties-start.opd top planar face.
    let corners = [(-4.0, -2.5), (0.0, -2.5), (0.0, 0.0), (-4.0, 0.0)];
    for (i, (x, y)) in corners.iter().enumerate() {
        geometry
            .add_point(SketchEntityId((i + 1) as u64), [*x, *y])
            .unwrap();
    }
    for i in 0..4 {
        let start = SketchEntityId((i + 1) as u64);
        let end = SketchEntityId(((i + 1) % 4 + 1) as u64);
        geometry
            .add_line(SketchEntityId((11 + i) as u64), [start, end])
            .unwrap();
    }
    let sketch = DocumentSketchId(1);
    snapshot
        .sketches
        .add_on_origin_plane(sketch, "Sketch1", WorkPlaneOrigin::XY, geometry)
        .unwrap();
    let selection = SketchPrismSelection {
        outer: (0..4)
            .map(|i| SketchDirectedCurve {
                curve: SketchEntityId((11 + i) as u64),
                reversed: false,
            })
            .collect(),
        holes: vec![],
    };
    let prism_def = SketchPrismDefinition::new(
        sketch,
        selection,
        [0.0, 0.0, 2.0].map(DatumLength::Constant),
    )
    .unwrap();
    let reads = prism_def.reads(&snapshot).unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(1),
            "Extrusion1",
            PartFeatureDefinition::SketchPrism(Box::new(prism_def)),
            reads,
        ))
        .unwrap();
    solve_selected_sketches(&mut snapshot);

    let history = owner(snapshot.clone());
    let doc = document();
    let mut evaluator = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let eval = snapshot.evaluate_part_features(&mut evaluator).unwrap();
    let solid = eval.body(FeatureId(1)).unwrap();
    let solid_key = solid.body_reference_key(&history, doc).unwrap().to_vec();
    let top_face = solid
        .face_reference_keys(&history, doc)
        .unwrap()
        .into_iter()
        .find(|fk| {
            solid
                .face_plane(&history, doc, fk)
                .is_ok_and(|plane| (plane.recipe().origin[2] - 2.0).abs() < 1e-4)
        })
        .expect("expected top planar face at z = 2.0 cm");

    // Create 1 mm (0.1 cm) offset surface (OffsetSrf1 -> Srf1).
    let offset_def = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        version: 1,
        target_body: solid_key,
        faces: vec![top_face],
        output: ThickenOutputMode::Surface,
        distance: 0.1,
        direction: ThickenDirection::Positive,
        boolean: ThickenBoolean::NewSolid,
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

    let history = owner(snapshot.clone());
    let mut evaluator = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let eval = snapshot.evaluate_part_features(&mut evaluator).unwrap();
    let srf = eval.body(FeatureId(2)).unwrap();
    assert!(srf.is_surface());
    let area = srf.surface_area_mm2(&history, doc).unwrap();
    assert!(
        (area - 1000.0).abs() < 1e-3,
        "expected 1000 mm^2, got {area}"
    );
    let srf_key = srf.body_reference_key(&history, doc).unwrap().to_vec();
    let edges = srf.edge_reference_keys(&history, doc).unwrap();
    assert_eq!(edges.len(), 4);
    (snapshot, srf_key, edges)
}

#[test]
fn extend_surface_single_edge_5mm_vs_10mm_and_stretch_vs_extend() {
    let (base, srf_key, edges) = block_with_offset_surface();
    let doc = document();
    let history = owner(base.clone());
    let mut evaluator = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let eval = base.evaluate_part_features(&mut evaluator).unwrap();
    let srf = eval.body(FeatureId(2)).unwrap();
    let long_edge = edges
        .iter()
        .find(|e| (srf.edge_length(&history, doc, e).unwrap() - 4.0).abs() < 1e-4)
        .expect("expected 40 mm edge")
        .clone();

    // 5 mm Extend (adds a 2nd extension face, total 1200 mm^2).
    let mut snap_5 = base.clone();
    let def_5 = ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
        version: 1,
        target_body: srf_key.clone(),
        edges: vec![long_edge.clone()],
        edge_chain: false,
        extent: ExtendSurfaceExtent::Distance(0.5),
        method: ExtendSurfaceMethod::Extend,
    })
    .unwrap();
    let reads = def_5.reads();
    snap_5
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Extend1",
            PartFeatureDefinition::ExtendSurface(Box::new(def_5)),
            reads,
        ))
        .unwrap();
    let hist_5 = owner(snap_5.clone());
    let mut eval_5 = CertifiedNamedSolidEvaluator::new(&hist_5, doc, certified_resolution());
    let part_5 = snap_5.evaluate_part_features(&mut eval_5).unwrap();
    let ext_5 = part_5.body(FeatureId(3)).unwrap();
    assert!(ext_5.is_surface());
    let area_5 = ext_5.surface_area_mm2(&hist_5, doc).unwrap();
    assert!(
        (area_5 - 1200.0).abs() < 1e-2,
        "expected 1200 mm^2 at 5 mm, got {area_5}"
    );
    assert_eq!(ext_5.face_reference_keys(&hist_5, doc).unwrap().len(), 2);

    // 10 mm Stretch (single enlarged face, total 1400 mm^2 > 1200 mm^2).
    let mut snap_10 = base;
    let def_10 = ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
        version: 1,
        target_body: srf_key,
        edges: vec![long_edge],
        edge_chain: false,
        extent: ExtendSurfaceExtent::Distance(1.0),
        method: ExtendSurfaceMethod::Stretch,
    })
    .unwrap();
    let reads = def_10.reads();
    snap_10
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Extend1",
            PartFeatureDefinition::ExtendSurface(Box::new(def_10)),
            reads,
        ))
        .unwrap();
    let hist_10 = owner(snap_10.clone());
    let mut eval_10 = CertifiedNamedSolidEvaluator::new(&hist_10, doc, certified_resolution());
    let part_10 = snap_10.evaluate_part_features(&mut eval_10).unwrap();
    let ext_10 = part_10.body(FeatureId(3)).unwrap();
    assert!(ext_10.is_surface());
    let area_10 = ext_10.surface_area_mm2(&hist_10, doc).unwrap();
    assert!(
        (area_10 - 1400.0).abs() < 1e-2,
        "expected 1400 mm^2 at 10 mm Stretch, got {area_10}"
    );
    assert_eq!(ext_10.face_reference_keys(&hist_10, doc).unwrap().len(), 1);
    assert!(area_10 > area_5);
    assert_eq!(
        part_10.ordered_bodies().filter(|b| b.is_surface()).count(),
        1,
        "Extend Surface replaces Srf1 in place"
    );
}

#[test]
fn extend_surface_edge_chain_and_to_plane_and_edit_round_trip() {
    let (mut snap, srf_key, edges) = block_with_offset_surface();
    let doc = document();
    let history = owner(snap.clone());
    let mut evaluator = CertifiedNamedSolidEvaluator::new(&history, doc, certified_resolution());
    let eval = snap.evaluate_part_features(&mut evaluator).unwrap();
    let srf = eval.body(FeatureId(2)).unwrap();

    // Edge Chain from any single edge returns all 4 rectangular perimeter edges.
    let chain = srf
        .surface_boundary_edge_chain(&history, doc, &edges[0], true)
        .unwrap();
    assert_eq!(chain.len(), 4);
    let single = srf
        .surface_boundary_edge_chain(&history, doc, &edges[0], false)
        .unwrap();
    assert_eq!(single.len(), 1);

    // 5 mm Stretch with Edge Chain expands 40x25 -> 50x35 = 1750 mm^2.
    let def_chain = ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
        version: 1,
        target_body: srf_key.clone(),
        edges: vec![edges[0].clone()],
        edge_chain: true,
        extent: ExtendSurfaceExtent::Distance(0.5),
        method: ExtendSurfaceMethod::Stretch,
    })
    .unwrap();
    let reads = def_chain.reads();
    snap.work_geometry
        .append(AuthoredFeature::new(
            FeatureId(3),
            "Extend1",
            PartFeatureDefinition::ExtendSurface(Box::new(def_chain)),
            reads,
        ))
        .unwrap();
    let hist = owner(snap.clone());
    let mut eval_chain = CertifiedNamedSolidEvaluator::new(&hist, doc, certified_resolution());
    let part = snap.evaluate_part_features(&mut eval_chain).unwrap();
    let ext = part.body(FeatureId(3)).unwrap();
    let area = ext.surface_area_mm2(&hist, doc).unwrap();
    assert!(
        (area - 1750.0).abs() < 1e-2,
        "expected 50x35 = 1750 mm^2 for Edge Chain, got {area}"
    );

    // Recipe serialization round-trip at program version 71.
    let program_recipe = snap.work_geometry.part_recipe().unwrap();
    assert_eq!(program_recipe.version, 71);
    let restored =
        oblikovati_model::feature::FeatureSequence::from_part_recipe(&program_recipe).unwrap();
    assert_eq!(restored.features().len(), 3);

    // Edit Extend1 to `To` XY plane (parallel to z = 2.1 cm surface -> kernel refuses cleanly).
    let to_def = ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
        version: 1,
        target_body: srf_key,
        edges: vec![edges[0].clone()],
        edge_chain: false,
        extent: ExtendSurfaceExtent::To(ExtendSurfaceTermination::OriginPlane(
            ExtendSurfaceOriginPlane::Xy,
        )),
        method: ExtendSurfaceMethod::Extend,
    })
    .unwrap();
    snap.edit_extend_surface(FeatureId(3), to_def).unwrap();
    let hist2 = owner(snap.clone());
    let mut eval2 = CertifiedNamedSolidEvaluator::new(&hist2, doc, certified_resolution());
    let part2 = snap.evaluate_part_features(&mut eval2).unwrap();
    assert!(part2.health().iter().any(|r| {
        r.feature == FeatureId(3) && matches!(r.status, FeatureEvaluationStatus::Failed(_))
    }));
}
