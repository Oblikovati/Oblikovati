// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Part Surface Replace Face (#3918).
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        AuthoredFeature,
        replace_face::{ReplaceFaceDefinition, ReplaceFaceRecipe, ReplaceFaceTarget},
        sketch_prism::{SketchDirectedCurve, SketchPrismSelection},
        sketch_revolve::SketchRevolveDefinition,
        thicken_offset::{
            ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean,
            ThickenDirection, ThickenOffsetDefinition, ThickenOffsetRecipe, ThickenOutputMode,
        },
        work_geometry::DatumAngle,
        work_plane::WorkPlaneOrigin,
    },
    sketch::{Sketch2D, SketchEntityId},
};

#[test]
fn replace_face_with_outward_offset_surface_extends_solid_by_exact_volume() {
    let mut history = owner(cube_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let initial = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let cube = initial.body(FeatureId(1)).expect("expected unit cube");
    let vol_before = cube.volume(&history, document()).unwrap();
    let body_key = cube
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let top_face = cube
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|face| {
            cube.face_topology(&history, document(), face)
                .unwrap()
                .surface
                == PrismSurfaceKind::Plane
                && (cube.face_bounds_center(&history, document(), face).unwrap()[2] - 1.0).abs()
                    < 1e-6
        })
        .expect("expected top planar face at Z = 1.0 cm");

    // Create OffsetSrf1 offsetting top face outward by 1 mm (0.1 cm).
    let offset_def = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        version: 1,
        target_body: body_key.clone(),
        faces: vec![top_face.clone()],
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
    let token = history.begin("Offset Surface").unwrap();
    let offset_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_thicken_offset("OffsetSrf1", offset_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let with_offset = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let offset_srf = with_offset
        .body(offset_id)
        .expect("expected OffsetSrf1 surface body");
    assert!(offset_srf.is_surface());
    let offset_body_key = offset_srf
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let offset_face_key = offset_srf
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .next()
        .expect("expected OffsetSrf1 face");

    // Replace top face with OffsetSrf1 face -> exact volume increase of 0.1 cm * 1.0 cm^2 = 0.1 cm^3.
    let replace_def = ReplaceFaceDefinition::from_recipe(ReplaceFaceRecipe {
        version: 1,
        body: body_key.clone(),
        existing_faces: vec![top_face],
        new_faces: vec![ReplaceFaceTarget {
            body: offset_body_key,
            face: offset_face_key,
        }],
        automatic_face_chain: false,
    })
    .unwrap();
    let token = history.begin("Replace Face").unwrap();
    let replace_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_replace_face("Replace Face1", replace_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let after = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        after
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        after.health()
    );
    let replaced = after
        .body(replace_id)
        .expect("expected replaced solid body");
    assert!(!replaced.is_surface(), "replaced body must remain a solid");
    assert!(
        after.body(offset_id).is_some(),
        "OffsetSrf1 surface body must remain in the part"
    );
    let vol_after = replaced.volume(&history, document()).unwrap();
    let expected = vol_before + 0.1;
    assert!(
        (vol_after - expected).abs() < 1e-6,
        "expected volume {expected} (1 mm * 1 cm^2 increase over {vol_before}), got {vol_after}"
    );
}

#[test]
fn replace_face_rejects_disjoint_torus_and_offset_surface_with_named_failure() {
    let mut snapshot = cube_document();
    // Add a separate toroidal solid via 360-degree revolve of a circle about an axis at X = 5.0 cm.
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(301), [7.0, 0.0]).unwrap();
    geometry
        .add_circle(SketchEntityId(302), SketchEntityId(301), 0.5)
        .unwrap();
    geometry
        .add_point(SketchEntityId(303), [5.0, -1.0])
        .unwrap();
    geometry.add_point(SketchEntityId(304), [5.0, 1.0]).unwrap();
    geometry
        .add_line(
            SketchEntityId(305),
            [SketchEntityId(303), SketchEntityId(304)],
        )
        .unwrap();
    geometry
        .set_line_centerline(SketchEntityId(305), true)
        .unwrap();
    let torus_sketch = DocumentSketchId(2);
    snapshot
        .sketches
        .add_on_origin_plane(torus_sketch, "TorusSketch", WorkPlaneOrigin::XZ, geometry)
        .unwrap();
    let selection = SketchPrismSelection {
        outer: vec![SketchDirectedCurve {
            curve: SketchEntityId(302),
            reversed: false,
        }],
        holes: vec![],
    };
    let rev_def = SketchRevolveDefinition::new(
        torus_sketch,
        selection,
        DatumAngle::constant(std::f64::consts::TAU).unwrap(),
    )
    .unwrap()
    .with_axis(SketchEntityId(305))
    .unwrap()
    .with_full_turn();
    let rev_reads = rev_def.reads(&snapshot).unwrap();
    let torus_id = FeatureId(2);
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            torus_id,
            "Revolution1",
            PartFeatureDefinition::SketchRevolve(Box::new(rev_def)),
            rev_reads,
        ))
        .unwrap();
    snapshot.sketches = snapshot
        .solve_document_sketches(Default::default(), || false)
        .candidate;

    let mut history = owner(snapshot);
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let before = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let cube = before.body(FeatureId(1)).expect("expected cube");
    let cube_body = cube
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let cube_top = cube
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|face| cube.face_bounds_center(&history, document(), face).unwrap()[2] > 0.99)
        .expect("expected cube top face");
    let torus = before
        .body(torus_id)
        .unwrap_or_else(|| panic!("expected toroidal solid; health={:?}", before.health()));
    let torus_body = torus
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let torus_face = torus
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .next()
        .expect("expected torus face");

    // Create offset surface 1 mm from the cube top face.
    let offset_def = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
        version: 1,
        target_body: cube_body,
        faces: vec![cube_top],
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
    let token = history.begin("Offset Surface").unwrap();
    let offset_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_thicken_offset("OffsetSrf1", offset_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let with_offset = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let offset_srf = with_offset.body(offset_id).expect("expected OffsetSrf1");
    let offset_body = offset_srf
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let offset_face = offset_srf
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .next()
        .expect("expected offset face");

    // Attempt to replace the torus face with the separate planar offset surface -> must fail cleanly.
    let replace_def = ReplaceFaceDefinition::from_recipe(ReplaceFaceRecipe {
        version: 1,
        body: torus_body,
        existing_faces: vec![torus_face],
        new_faces: vec![ReplaceFaceTarget {
            body: offset_body,
            face: offset_face,
        }],
        automatic_face_chain: false,
    })
    .unwrap();
    let token = history.begin("Replace Face").unwrap();
    let replace_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_replace_face("Replace Face1", replace_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let after = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(after.body(replace_id).is_none());
    assert!(
        after.body(torus_id).is_some(),
        "failed replace face must preserve original torus solid"
    );
    let status = format!(
        "{:?}",
        after
            .health()
            .iter()
            .find(|record| record.feature == replace_id)
            .expect("expected Replace Face health record")
            .status
    );
    assert!(
        status.contains("Create Replace Face feature failed"),
        "expected named Replace Face failure, saw {status}"
    );
}
