// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Surface Trim (plane split, region flip, analytic area reduction, #3918).
use super::*;
use oblikovati_martensite::PrismMeshOptions;
use oblikovati_model::feature::{
    hole::HoleOriginPlane,
    surface_trim::{
        ResolvedSurfaceTrimTool, SurfaceTrimDefinition, SurfaceTrimRecipe, SurfaceTrimSide,
        SurfaceTrimTool,
    },
    thicken_offset::{
        ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean, ThickenDirection,
        ThickenOffsetDefinition, ThickenOffsetRecipe, ThickenOutputMode,
    },
    work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference},
};
use oblikovati_model::sketch::SketchPlane;

fn find_vertical_side_face(
    history: &DocumentEditHistory,
    eval: &oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
) -> (Vec<u8>, Vec<u8>) {
    // Base selected_prism_document() is a 4 x 3 x 2 cm block spanning x in [0, 4], y in [0, 3], z in [0, 2].
    // The vertical x = 4 side face has area 3 x 2 = 6 cm^2 and spans y in [0, 3].
    let body = eval.ordered_bodies().next().unwrap();
    let body_key = body
        .body_reference_key(history, document())
        .unwrap()
        .to_vec();
    for face_key in body.face_reference_keys(history, document()).unwrap() {
        let center = body
            .face_bounds_center(history, document(), &face_key)
            .unwrap();
        if (center[0] - 4.0).abs() < 1e-3 && (center[2] - 1.0).abs() < 1e-3 {
            return (body_key, face_key);
        }
    }
    panic!("expected vertical x=4 side face on base block");
}

#[test]
fn surface_trim_splits_surface_reduces_area_and_supports_region_flip() {
    let mut history = owner(selected_prism_document());
    let base_eval = history
        .evaluation_snapshot()
        .snapshot()
        .evaluate_part_features(&mut CertifiedNamedSolidEvaluator::new(
            &history,
            document(),
            certified_resolution(),
        ))
        .unwrap();
    let (solid_body_key, side_face_key) = find_vertical_side_face(&history, &base_eval);

    // Create zero-offset surface from the vertical side face (3 cm x 2 cm = 6.0 cm^2).
    let offset_def = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        version: 1,
        target_body: solid_body_key,
        faces: vec![side_face_key],
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
    })
    .unwrap();
    let token = history.begin("OffsetSrf1").unwrap();
    let offset_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_thicken_offset("OffsetSrf1", offset_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    // Add a Work Plane parallel to XZ at y = 1.5 (bisecting y in [0, 3]).
    // SketchPlane::xz() has normal (0, -1, 0), so offset -1.5 places it at y = +1.5.
    let token = history.begin("MidXzPlane").unwrap();
    let plane_id = history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate_feature().unwrap();
            let def = WorkPlaneDefinition::Offset {
                base: WorkPlaneReference::Origin(WorkPlaneOrigin::XZ),
                distance: WorkPlaneDistance::Constant(-1.5),
            };
            let reads = def.reads();
            snapshot
                .work_geometry
                .append(oblikovati_model::feature::AuthoredFeature::new(
                    id,
                    "Work Plane1",
                    def,
                    reads,
                ))
                .unwrap();
            Ok::<_, oblikovati_model::history::HistoryError>(id)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let before_eval = history
        .evaluation_snapshot()
        .snapshot()
        .evaluate_part_features(&mut CertifiedNamedSolidEvaluator::new(
            &history,
            document(),
            certified_resolution(),
        ))
        .unwrap();
    let surface_body = before_eval.body(offset_id).expect("expected surface body");
    assert!(surface_body.is_surface());
    let surface_key = surface_body
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let before_faces = surface_body
        .face_reference_keys(&history, document())
        .unwrap();
    let before_area: f64 = before_faces
        .iter()
        .map(|f| surface_body.face_area(&history, document(), f).unwrap())
        .sum();
    assert!(
        (before_area - 6.0).abs() < 1e-4,
        "before_area={before_area}"
    );

    // Verify interactive preview splits into two equal 3.0 cm^2 halves on both sides of Work Plane1.
    let cut_plane = SketchPlane::from_origin_normal(
        [0.0, 1.5, 0.0],
        [0.0, -1.0, 0.0],
        oblikovati_model::sketch::SketchPlaneOptions::default(),
    )
    .unwrap();
    let preview = surface_body
        .preview_surface_trim_regions(
            &history,
            document(),
            None,
            &ResolvedSurfaceTrimTool::Plane(cut_plane),
            PrismMeshOptions {
                chord_error: 0.05,
                angular_error: 0.2,
            },
        )
        .unwrap();
    assert!((preview.positive_area - 3.0).abs() < 1e-4);
    assert!((preview.negative_area - 3.0).abs() < 1e-4);
    assert!(!preview.positive_triangles.is_empty());
    assert!(!preview.negative_triangles.is_empty());

    // Trim removing Negative side (y > 1.5), keeping y in [0, 1.5].
    let trim_neg = SurfaceTrimDefinition::from_recipe(SurfaceTrimRecipe {
        version: 1,
        tool: SurfaceTrimTool::WorkPlane(plane_id),
        target_body: surface_key.clone(),
        remove_side: SurfaceTrimSide::Negative,
    })
    .unwrap();
    let token = history.begin("Trim1").unwrap();
    let trim_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_surface_trim("Trim1", trim_neg, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let trimmed_neg = history
        .evaluation_snapshot()
        .snapshot()
        .evaluate_part_features(&mut CertifiedNamedSolidEvaluator::new(
            &history,
            document(),
            certified_resolution(),
        ))
        .unwrap();
    let body_neg = trimmed_neg.body(trim_id).expect("expected trimmed surface");
    assert!(body_neg.is_surface());
    let neg_faces = body_neg.face_reference_keys(&history, document()).unwrap();
    let neg_area: f64 = neg_faces
        .iter()
        .map(|f| body_neg.face_area(&history, document(), f).unwrap())
        .sum();
    assert!((neg_area - 3.0).abs() < 1e-4, "neg_area={neg_area}");
    let neg_center = body_neg
        .face_bounds_center(&history, document(), &neg_faces[0])
        .unwrap();
    assert!(
        neg_center[1] < 1.5,
        "expected y < 1.5 half kept, got {neg_center:?}"
    );

    // Edit Trim1 flipping remove_side to Positive (y < 1.5), keeping y in [1.5, 3.0].
    let trim_pos = SurfaceTrimDefinition::from_recipe(SurfaceTrimRecipe {
        version: 1,
        tool: SurfaceTrimTool::WorkPlane(plane_id),
        target_body: surface_key.clone(),
        remove_side: SurfaceTrimSide::Positive,
    })
    .unwrap();
    let token = history.begin("EditTrim1").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.edit_surface_trim(trim_id, trim_pos)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let trimmed_pos = history
        .evaluation_snapshot()
        .snapshot()
        .evaluate_part_features(&mut CertifiedNamedSolidEvaluator::new(
            &history,
            document(),
            certified_resolution(),
        ))
        .unwrap();
    let body_pos = trimmed_pos
        .body(trim_id)
        .expect("expected flipped trimmed surface");
    assert!(body_pos.is_surface());
    let pos_faces = body_pos.face_reference_keys(&history, document()).unwrap();
    let pos_area: f64 = pos_faces
        .iter()
        .map(|f| body_pos.face_area(&history, document(), f).unwrap())
        .sum();
    assert!((pos_area - 3.0).abs() < 1e-4, "pos_area={pos_area}");
    let pos_center = body_pos
        .face_bounds_center(&history, document(), &pos_faces[0])
        .unwrap();
    assert!(
        pos_center[1] > 1.5,
        "expected y > 1.5 half kept, got {pos_center:?}"
    );

    // Non-intersecting origin plane (XZ at y=0 is on the boundary, does not bisect y in [0,3]) must fail cleanly.
    let bad_trim = SurfaceTrimDefinition::from_recipe(SurfaceTrimRecipe {
        version: 1,
        tool: SurfaceTrimTool::OriginPlane(HoleOriginPlane::Xz),
        target_body: surface_key,
        remove_side: SurfaceTrimSide::Positive,
    })
    .unwrap();
    let token = history.begin("BadTrim").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.edit_surface_trim(trim_id, bad_trim)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let failed_eval = history
        .evaluation_snapshot()
        .snapshot()
        .evaluate_part_features(&mut CertifiedNamedSolidEvaluator::new(
            &history,
            document(),
            certified_resolution(),
        ))
        .unwrap();
    assert!(
        failed_eval.body(trim_id).is_none(),
        "expected non-crossing XZ plane trim to fail cleanly"
    );
}
