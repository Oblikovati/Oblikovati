// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Mark eligibility, face chaining, zero-volume rejection, and replay (#3918).
use super::*;
use oblikovati_martensite::{MarkEligibility, check_mark_eligibility};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::mark::{
        MarkDefinition, MarkDirection, MarkGeometrySet, MarkMethod, MarkRecipe, MarkType,
    },
    sketch::{SketchEntityId, SketchPlane, SketchPlaneOptions},
};

fn add_open_line_sketch(history: &mut DocumentEditHistory) -> (DocumentSketchId, SketchEntityId) {
    let token = history.begin("Mark sketch").unwrap();
    let result = history
        .edit(&token, |snapshot, ids| {
            let sketch_id = ids.allocate_sketch()?;
            let plane = SketchPlane::from_recipe(
                oblikovati_model::sketch::SketchPlaneRecipe {
                    origin: [0.0, 0.0, 1.0],
                    x_axis: [1.0, 0.0, 0.0],
                    y_axis: [0.0, 1.0, 0.0],
                },
                SketchPlaneOptions::default(),
            )
            .unwrap();
            let mut geometry = oblikovati_model::sketch::Sketch2D::default();
            let p0 = ids.allocate_sketch_entity()?;
            let p1 = ids.allocate_sketch_entity()?;
            let line = ids.allocate_sketch_entity()?;
            geometry.add_point(p0, [0.2, 0.5]).unwrap();
            geometry.add_point(p1, [0.8, 0.5]).unwrap();
            geometry.add_line(line, [p0, p1]).unwrap();
            snapshot
                .sketches
                .add(sketch_id, "Sketch2", plane, geometry)
                .unwrap();
            solve_selected_sketches(snapshot);
            Ok::<_, oblikovati_model::history::HistoryError>((sketch_id, line))
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    result
}

#[test]
fn mark_eligibility_distinguishes_zero_single_and_multiple_solids() {
    let history = owner(cube_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let eval = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let bodies: Vec<_> = eval.ordered_bodies().collect();
    assert_eq!(
        check_mark_eligibility(&history, document(), std::iter::empty()),
        MarkEligibility::NoVolume
    );
    assert_eq!(
        check_mark_eligibility(&history, document(), bodies.iter().copied()),
        MarkEligibility::Eligible
    );
    assert_eq!(
        check_mark_eligibility(&history, document(), [bodies[0], bodies[0]].into_iter()),
        MarkEligibility::MultipleSolidBodies { count: 2 }
    );
}

#[test]
fn mark_surface_and_mark_through_replay_preserve_volume_and_produce_overlays() {
    let mut history = owner(cube_document());
    let (sketch_id, line_id) = add_open_line_sketch(&mut history);
    let body_key = key("body", &[1]);
    let recipe = MarkRecipe {
        version: 1,
        body: body_key.clone(),
        geometry_sets: vec![MarkGeometrySet {
            sketch: sketch_id,
            curves: vec![line_id],
            mark_type: MarkType::MarkSurface,
        }],
        method: MarkMethod::ProjectToClosestFace,
        faces: Vec::new(),
        automatic_face_chain: false,
        direction: MarkDirection::Positive,
    };
    let token = history.begin("Mark").unwrap();
    let mark_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_mark(
                "Mark1",
                MarkDefinition::from_recipe(recipe.clone()).unwrap(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let eval_surface = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap()
    };
    let surface_body = eval_surface.body(mark_id).expect("expected Mark1 body");
    assert!((surface_body.volume(&history, document()).unwrap() - 1.0).abs() < 1e-6);
    let surface_count = surface_body.mark_overlay_triangles().len();
    assert!(surface_count > 0, "expected Mark Surface imprint triangles");

    // Verify automatic connected face chain on the cube reaches all 6 faces.
    let face_keys = surface_body
        .face_reference_keys(&history, document())
        .unwrap();
    assert_eq!(face_keys.len(), 6);
    let chained = surface_body
        .connected_face_chain(&history, document(), &face_keys[0])
        .unwrap();
    assert_eq!(chained.len(), 6);

    // Edit Mark1 to Mark Through with Wrap to Face across chained faces.
    let through_recipe = MarkRecipe {
        version: 1,
        body: body_key,
        geometry_sets: vec![MarkGeometrySet {
            sketch: sketch_id,
            curves: vec![line_id],
            mark_type: MarkType::MarkThrough,
        }],
        method: MarkMethod::WrapToFace,
        faces: chained,
        automatic_face_chain: true,
        direction: MarkDirection::Positive,
    };
    let token = history.begin("Edit Mark").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.edit_mark(
                mark_id,
                MarkDefinition::from_recipe(through_recipe).unwrap(),
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let eval_through = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap()
    };
    let through_body = eval_through.body(mark_id).expect("expected edited Mark1");
    assert!((through_body.volume(&history, document()).unwrap() - 1.0).abs() < 1e-6);
    assert!(
        through_body.mark_overlay_triangles().len() > surface_count,
        "expected Mark Through to add opposing face imprint triangles"
    );
}

#[test]
fn mark_rejects_surface_only_zero_volume_body() {
    use oblikovati_model::feature::unwrap::{UnwrapAlignment, UnwrapDefinition, UnwrapRecipe};
    let mut history = owner(cube_document());
    let (sketch_id, line_id) = add_open_line_sketch(&mut history);
    let face_key = key("end", &[1]);
    let unwrap_def = UnwrapDefinition::from_recipe(&UnwrapRecipe {
        version: 1,
        target_body: key("body", &[1]),
        faces: vec![face_key],
        alignment_vertex: None,
        auto_face_chain: false,
        merge_result_body: true,
        alignment: UnwrapAlignment::OriginXY,
        linear_result_edges: Vec::new(),
        rigid_result_edges: Vec::new(),
        seam_edges: Vec::new(),
    })
    .unwrap();
    let token = history.begin("Unwrap").unwrap();
    let unwrap_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_unwrap("Unwrap1", unwrap_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let eval_unwrap = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap()
    };
    let surface = eval_unwrap
        .body(unwrap_id)
        .expect("expected Unwrap1 surface");
    assert!(surface.is_surface());
    assert_eq!(
        check_mark_eligibility(&history, document(), std::iter::once(surface)),
        MarkEligibility::NoVolume
    );

    let surface_body_key = surface
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let bad_mark = MarkDefinition::from_recipe(MarkRecipe {
        version: 1,
        body: surface_body_key,
        geometry_sets: vec![MarkGeometrySet {
            sketch: sketch_id,
            curves: vec![line_id],
            mark_type: MarkType::MarkSurface,
        }],
        method: MarkMethod::ProjectToClosestFace,
        faces: Vec::new(),
        automatic_face_chain: false,
        direction: MarkDirection::Positive,
    })
    .unwrap();
    let token = history.begin("Mark surface").unwrap();
    let mark_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_mark("Mark1", bad_mark, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let eval_bad = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap()
    };
    let record = eval_bad
        .health()
        .iter()
        .find(|entry| entry.feature == mark_id)
        .expect("expected Mark1 health record");
    let diagnostic = format!("{:?}", record.status);
    assert!(
        diagnostic.contains("Mark failed. The selected component has no volume."),
        "expected no-volume diagnostic, saw {diagnostic}"
    );
}
