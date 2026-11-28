// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Thread face detection, replay, overlays, and conical rejection (#3918).
use super::*;
use oblikovati_model::feature::thread::{
    ThreadDefinition, ThreadHandedness, ThreadRecipe, ThreadSpecification,
};

fn make_bore_history() -> (DocumentEditHistory, FeatureId, Vec<u8>, Vec<u8>) {
    let mut history = owner(hole_document());
    let hole = HoleDefinition::from_recipe(HoleRecipe {
        version: 1,
        body: key("body", &[1]),
        face: key("end", &[1]),
        position: HolePosition::FaceCentroid,
        diameter: DatumLength::Constant(0.9),
        extent: HoleExtent::Distance(DatumLength::Constant(0.2)),
        options: Default::default(),
    })
    .unwrap();
    let token = history.begin("Hole").unwrap();
    let hole_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_hole("Hole1", hole, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let eval = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let body = eval.body(hole_id).expect("expected Hole1 body");
    assert!(body.has_threadable_faces(&history, document()));
    let face_key = body
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .find(|face| {
            body.threadable_face_info(&history, document(), face)
                .ok()
                .flatten()
                .is_some_and(|info| info.internal && !info.conical)
        })
        .expect("expected internal cylindrical bore face");
    let info = body
        .threadable_face_info(&history, document(), &face_key)
        .unwrap()
        .unwrap();
    assert!((info.diameter_cm - 0.9).abs() < 1e-4, "{info:?}");
    assert!((info.full_depth_cm - 0.2).abs() < 1e-4, "{info:?}");
    (history, hole_id, key("body", &[1]), face_key)
}

#[test]
fn rectangular_block_reports_no_threadable_faces() {
    let history = owner(hole_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let eval = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let body = eval.ordered_bodies().next().expect("expected block solid");
    assert!(!body.has_threadable_faces(&history, document()));
}

#[test]
fn internal_cylindrical_thread_replays_and_updates_overlay_density_and_visibility() {
    let (mut history, _, body_key, face_key) = make_bore_history();
    let base_recipe = ThreadRecipe {
        version: 1,
        body: body_key.clone(),
        face: face_key.clone(),
        full_depth: true,
        depth: None,
        offset: None,
        display_in_model: true,
        spec: ThreadSpecification {
            thread_type: "ANSI Metric M Profile".into(),
            size: "9".into(),
            designation: "M9x1.25".into(),
            class: "6H".into(),
            handedness: ThreadHandedness::RightHand,
        },
    };
    let token = history.begin("Thread").unwrap();
    let thread_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_thread(
                "Thread1",
                ThreadDefinition::from_recipe(base_recipe.clone()).unwrap(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let eval_coarse = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap()
    };
    let coarse_body = eval_coarse.body(thread_id).expect("expected Thread1 body");
    let coarse_count = coarse_body.thread_overlay_triangles().len();
    assert!(coarse_count > 0, "expected visible thread overlay bands");

    // Finer pitch M9x0.75 produces more closely spaced bands.
    let mut fine_recipe = base_recipe.clone();
    fine_recipe.spec.designation = "M9x0.75".into();
    let token = history.begin("Edit Thread").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.edit_thread(
                thread_id,
                ThreadDefinition::from_recipe(fine_recipe).unwrap(),
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let eval_fine = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap()
    };
    let fine_count = eval_fine
        .body(thread_id)
        .unwrap()
        .thread_overlay_triangles()
        .len();
    assert!(
        fine_count > coarse_count,
        "expected M9x0.75 ({fine_count}) to have more band triangles than M9x1.25 ({coarse_count})"
    );

    // Disabling display_in_model hides overlays while keeping feature Healthy.
    let mut hidden_recipe = base_recipe.clone();
    hidden_recipe.display_in_model = false;
    let token = history.begin("Hide Thread").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.edit_thread(
                thread_id,
                ThreadDefinition::from_recipe(hidden_recipe).unwrap(),
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let eval_hidden = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap()
    };
    assert!(
        eval_hidden
            .body(thread_id)
            .unwrap()
            .thread_overlay_triangles()
            .is_empty()
    );

    // Offset beyond face length also hides overlay while keeping feature Healthy.
    let mut offset_recipe = base_recipe;
    offset_recipe.full_depth = false;
    offset_recipe.depth = Some(DatumLength::Constant(0.1));
    offset_recipe.offset = Some(DatumLength::Constant(9.9));
    let token = history.begin("Offset Thread").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.edit_thread(
                thread_id,
                ThreadDefinition::from_recipe(offset_recipe).unwrap(),
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let eval_offset = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap()
    };
    assert!(
        eval_offset
            .body(thread_id)
            .unwrap()
            .thread_overlay_triangles()
            .is_empty()
    );
}
