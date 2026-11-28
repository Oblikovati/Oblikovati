// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Combine Join, multi-tool Cut, Intersect failure, and Keep Toolbodies (#3918).
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        AuthoredFeature,
        combine::{CombineDefinition, CombineOperation, CombineRecipe},
        sketch_prism::{SketchDirectedCurve, SketchPrismSelection},
        work_plane::WorkPlaneOrigin,
    },
    sketch::{Sketch2D, SketchEntityId},
};

#[allow(clippy::too_many_arguments)]
fn add_cylinder_extrusion(
    snapshot: &mut DocumentSnapshot,
    sketch_id: u64,
    center_id: u64,
    circle_id: u64,
    center: [f64; 2],
    radius: f64,
    height: f64,
    feature_id: u64,
    name: &str,
) {
    let mut geometry = Sketch2D::default();
    geometry
        .add_point(SketchEntityId(center_id), center)
        .unwrap();
    geometry
        .add_circle(SketchEntityId(circle_id), SketchEntityId(center_id), radius)
        .unwrap();
    let sketch = DocumentSketchId(sketch_id);
    snapshot
        .sketches
        .add_on_origin_plane(
            sketch,
            &format!("ToolSketch{sketch_id}"),
            WorkPlaneOrigin::XY,
            geometry,
        )
        .unwrap();
    let selection = SketchPrismSelection {
        outer: vec![SketchDirectedCurve {
            curve: SketchEntityId(circle_id),
            reversed: false,
        }],
        holes: vec![],
    };
    let definition = SketchPrismDefinition::new(
        sketch,
        selection,
        [0.0, 0.0, height].map(DatumLength::Constant),
    )
    .unwrap();
    let reads = definition.reads(snapshot).unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(feature_id),
            name,
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        ))
        .unwrap();
}

fn three_body_combine_document() -> DocumentSnapshot {
    // 4 x 3 x 2 cm block (volume 24 cm^3) + two cylinders at (1, 1.5) and (3, 1.5), r=0.5, h=3 cm.
    let mut snapshot = selected_prism_document();
    add_cylinder_extrusion(
        &mut snapshot,
        2,
        201,
        202,
        [1.0, 1.5],
        0.5,
        3.0,
        2,
        "Extrusion1",
    );
    add_cylinder_extrusion(
        &mut snapshot,
        3,
        301,
        302,
        [3.0, 1.5],
        0.5,
        3.0,
        3,
        "Extrusion2",
    );
    solve_selected_sketches(&mut snapshot);
    snapshot
}

fn body_key_for(feature: u64, sketch: u64) -> Vec<u8> {
    SemanticReferenceName::new(document(), feature, 4, "sketch-prism", "body", &[sketch])
        .unwrap()
        .encode()
        .unwrap()
}

#[test]
fn combine_join_keep_toolbodies_and_multi_tool_cut_match_analytic_volumes() {
    let pi = std::f64::consts::PI;

    // 1. Join block + cyl1 with keep_toolbodies = true -> 3 solids total (cyl1 retained), volume = 24 + pi * 0.25 * 1.0
    let mut history = owner(three_body_combine_document());
    let base_key = body_key_for(1, 1);
    let cyl1_key = body_key_for(2, 2);
    let cyl2_key = body_key_for(3, 3);
    let join_def = CombineDefinition::from_recipe(CombineRecipe {
        version: 1,
        base_body: base_key.clone(),
        tool_bodies: vec![cyl1_key.clone()],
        operation: CombineOperation::Join,
        keep_toolbodies: true,
    })
    .unwrap();
    let token = history.begin("Combine").unwrap();
    let join_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_combine("Combine1", join_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let join_eval = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        join_eval
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "join health: {:?}",
        join_eval.health()
    );
    assert_eq!(join_eval.ordered_bodies().count(), 3);
    let joined = join_eval.body(join_id).expect("expected joined solid");
    let expected_join = 24.0 + pi * 0.25 * 1.0;
    let actual_join = joined.volume(&history, document()).unwrap();
    assert!(
        (actual_join - expected_join).abs() < 1e-4,
        "Join volume {actual_join}, expected {expected_join}"
    );
    assert!(join_eval.body(FeatureId(2)).is_some());

    // 2. Multi-tool Cut (block - cyl1 - cyl2) with keep_toolbodies = false -> 1 solid!
    let mut history = owner(three_body_combine_document());
    let cut_def = CombineDefinition::from_recipe(CombineRecipe {
        version: 1,
        base_body: base_key.clone(),
        tool_bodies: vec![cyl1_key.clone(), cyl2_key.clone()],
        operation: CombineOperation::Cut,
        keep_toolbodies: false,
    })
    .unwrap();
    let token = history.begin("Combine").unwrap();
    let cut_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_combine("Combine1", cut_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let cut_eval = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        cut_eval
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "cut health: {:?}",
        cut_eval.health()
    );
    assert_eq!(cut_eval.ordered_bodies().count(), 1);
    let cut_body = cut_eval.body(cut_id).expect("expected cut solid");
    let expected_cut = 24.0 - 2.0 * (pi * 0.25 * 2.0);
    let actual_cut = cut_body.volume(&history, document()).unwrap();
    assert!(
        (actual_cut - expected_cut).abs() < 1e-4,
        "Multi-tool Cut volume {actual_cut}, expected {expected_cut}"
    );

    // 3. Intersect block with two disjoint cylinders -> cleanly fails (not a single solid)
    let mut history = owner(three_body_combine_document());
    let intersect_def = CombineDefinition::from_recipe(CombineRecipe {
        version: 1,
        base_body: base_key.clone(),
        tool_bodies: vec![cyl1_key.clone(), cyl2_key.clone()],
        operation: CombineOperation::Intersect,
        keep_toolbodies: false,
    })
    .unwrap();
    let token = history.begin("Combine").unwrap();
    let intersect_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_combine("Combine1", intersect_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let intersect_eval = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        matches!(
            intersect_eval
                .health()
                .iter()
                .find(|record| record.feature == intersect_id)
                .map(|record| &record.status),
            Some(FeatureEvaluationStatus::Failed(_))
        ),
        "expected Intersect with two disjoint cylinders to fail cleanly: {:?}",
        intersect_eval.health()
    );

    // 4. Single-tool Cut(block, cyl1) with keep_toolbodies = true -> exact volume 24 - 0.5*pi and cyl1 retained
    let mut history = owner(three_body_combine_document());
    let cut1_def = CombineDefinition::from_recipe(CombineRecipe {
        version: 1,
        base_body: base_key,
        tool_bodies: vec![cyl1_key],
        operation: CombineOperation::Cut,
        keep_toolbodies: true,
    })
    .unwrap();
    let token = history.begin("Combine").unwrap();
    let cut1_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_combine("Combine1", cut1_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let cut1_eval = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        cut1_eval
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "single-tool cut health: {:?}",
        cut1_eval.health()
    );
    assert_eq!(cut1_eval.ordered_bodies().count(), 3);
    assert!(cut1_eval.body(FeatureId(2)).is_some());
    let cut1_body = cut1_eval
        .body(cut1_id)
        .expect("expected single-tool cut solid");
    let expected_cut1 = 24.0 - pi * 0.25 * 2.0;
    let actual_cut1 = cut1_body.volume(&history, document()).unwrap();
    assert!(
        (actual_cut1 - expected_cut1).abs() < 1e-4,
        "Single-tool Cut volume {actual_cut1}, expected {expected_cut1}"
    );
}
