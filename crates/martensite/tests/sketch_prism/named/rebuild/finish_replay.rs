// SPDX-License-Identifier: GPL-2.0-only
//! Martensite unit and ordered replay tests for Part Modify Finish (#3918).
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, finish_appearance_rgba};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        AuthoredFeature, FeatureEvaluationStatus,
        finish::{FinishDefinition, FinishRecipe, FinishType},
        sketch_prism::{SketchDirectedCurve, SketchPrismSelection},
        work_plane::WorkPlaneOrigin,
    },
    sketch::{Sketch2D, SketchEntityId},
};

fn rect_40x25x15_document() -> DocumentSnapshot {
    let mut snapshot = selected_prism_document();
    snapshot.work_geometry = Default::default();
    snapshot.sketches = Default::default();
    let mut geometry = Sketch2D::default();
    let corners = [(0.0, 0.0), (4.0, 0.0), (4.0, 2.5), (0.0, 2.5)];
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
    let definition = SketchPrismDefinition::new(
        sketch,
        selection,
        [0.0, 0.0, 1.5].map(DatumLength::Constant),
    )
    .unwrap();
    let reads = definition.reads(&snapshot).unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(1),
            "Extrusion1",
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        ))
        .unwrap();
    solve_selected_sketches(&mut snapshot);
    snapshot
}

fn add_second_cylinder_body(snapshot: &mut DocumentSnapshot) -> FeatureId {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(201), [1.0, 1.5]).unwrap();
    geometry
        .add_circle(SketchEntityId(202), SketchEntityId(201), 0.5)
        .unwrap();
    let sketch = DocumentSketchId(2);
    snapshot
        .sketches
        .add_on_origin_plane(sketch, "Sketch2", WorkPlaneOrigin::XY, geometry)
        .unwrap();
    let selection = SketchPrismSelection {
        outer: vec![SketchDirectedCurve {
            curve: SketchEntityId(202),
            reversed: false,
        }],
        holes: vec![],
    };
    let definition = SketchPrismDefinition::new(
        sketch,
        selection,
        [0.0, 0.0, 2.0].map(DatumLength::Constant),
    )
    .unwrap();
    let reads = definition.reads(snapshot).unwrap();
    let cyl_id = FeatureId(2);
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            cyl_id,
            "Extrusion2",
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        ))
        .unwrap();
    solve_selected_sketches(snapshot);
    cyl_id
}

#[test]
fn finish_computes_exact_1000_mm2_area_and_attaches_face_appearance_color() {
    let mut history = owner(rect_40x25x15_document());
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let before = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let solid = before.body(FeatureId(1)).unwrap();
    let body_key = solid
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let faces = solid.face_reference_keys(&history, document()).unwrap();
    let top_face = faces
        .iter()
        .find(|key| {
            solid
                .face_bounds_center(&history, document(), key)
                .ok()
                .is_some_and(|c| (c[2] - 1.5).abs() < 1e-4)
        })
        .expect("expected top planar face at z = 1.5 cm")
        .clone();
    let bottom_face = faces
        .iter()
        .find(|key| {
            solid
                .face_bounds_center(&history, document(), key)
                .ok()
                .is_some_and(|c| c[2].abs() < 1e-4)
        })
        .expect("expected bottom planar face at z = 0 cm")
        .clone();

    assert_eq!(
        solid
            .finish_area_mm2(&history, document(), &[], &[])
            .unwrap(),
        0.0
    );
    let top_area = solid
        .finish_area_mm2(&history, document(), std::slice::from_ref(&top_face), &[])
        .unwrap();
    assert!(
        (top_area - 1000.0).abs() < 1e-3,
        "expected 40x25 mm top face area to be 1000 mm², got {top_area}"
    );
    let net_area = solid
        .finish_area_mm2(
            &history,
            document(),
            &[top_face.clone(), bottom_face.clone()],
            &[bottom_face],
        )
        .unwrap();
    assert!(
        (net_area - 1000.0).abs() < 1e-3,
        "expected excluded face area subtraction to yield 1000 mm², got {net_area}"
    );
    let before_vol = solid.volume(&history, document()).unwrap();

    let definition = FinishDefinition::from_recipe(FinishRecipe {
        version: 1,
        body: body_key,
        included_faces: vec![top_face.clone()],
        excluded_faces: Vec::new(),
        finish_type: FinishType::Paint,
        process: "Prime/Paint RAL 3018".into(),
        appearance: "Red".into(),
        thickness: "50 micron".into(),
        hardness: String::new(),
        depth: String::new(),
        description: String::new(),
        short_description: String::new(),
        comment: String::new(),
        export_all_parameters: true,
    })
    .unwrap();
    let token = history.begin("Finish").unwrap();
    let finish_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_finish("Finish1", definition, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let after = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let finished = after.body(finish_id).unwrap();
    let after_vol = finished.volume(&history, document()).unwrap();
    assert!(
        (after_vol - before_vol).abs() < 1e-6,
        "expected non-destructive finish to preserve solid volume"
    );
    let colors = finished
        .finish_face_colors()
        .expect("expected finish color");
    assert_eq!(colors.get(&top_face), Some(&finish_appearance_rgba("Red")));

    // Round-trip recipe version contract (program version 65, source version 55).
    let recipe = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 65);
    let finish_recipe = recipe.features.iter().find(|f| f.id == finish_id).unwrap();
    assert_eq!(finish_recipe.definition.version, 55);
    let restored = oblikovati_model::feature::FeatureSequence::from_part_recipe(&recipe).unwrap();
    assert_eq!(restored.features().len(), 2);
}

#[test]
fn finish_reports_blocked_warning_when_upstream_body_is_suppressed() {
    let mut snapshot = rect_40x25x15_document();
    let cyl_id = add_second_cylinder_body(&mut snapshot);
    let mut history = owner(snapshot);
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let before = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let cyl_solid = before.body(cyl_id).unwrap();
    let cyl_body_key = cyl_solid
        .body_reference_key(&history, document())
        .unwrap()
        .to_vec();
    let cyl_face = cyl_solid
        .face_reference_keys(&history, document())
        .unwrap()
        .into_iter()
        .next()
        .expect("expected cylinder face");

    let definition = FinishDefinition::from_recipe(FinishRecipe {
        version: 1,
        body: cyl_body_key,
        included_faces: vec![cyl_face],
        excluded_faces: Vec::new(),
        finish_type: FinishType::Paint,
        process: "Prime/Paint RAL 3018".into(),
        appearance: "Red".into(),
        thickness: "50 micron".into(),
        hardness: String::new(),
        depth: String::new(),
        description: String::new(),
        short_description: String::new(),
        comment: String::new(),
        export_all_parameters: true,
    })
    .unwrap();
    let token = history.begin("Finish").unwrap();
    let finish_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_finish("Finish1", definition, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    // Suppress the cylindrical body's extrusion while leaving the block visible.
    let token = history.begin("Suppress Cylinder").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.set_feature_suppressed(cyl_id, true)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let after = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    assert!(after.body(FeatureId(1)).is_some(), "block must stay live");
    assert!(after.body(cyl_id).is_none(), "cylinder must be suppressed");
    let finish_health = after
        .health()
        .iter()
        .find(|h| h.feature == finish_id)
        .expect("expected Finish1 health record");
    assert!(
        matches!(
            &finish_health.status,
            FeatureEvaluationStatus::Failed(
                oblikovati_model::feature::part::PartFeatureFailure::BodyReference(_)
            ) | FeatureEvaluationStatus::Blocked { .. }
        ),
        "expected Finish1 warning/missing-geometry health when cylinder is suppressed, got {:?}",
        finish_health.status
    );
}
