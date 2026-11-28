// SPDX-License-Identifier: GPL-2.0-only
//! Real-kernel verification of Part Modify Split (AllFaces, single face, and Solid Both/SideA/SideB, #3918).
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        split::{SplitDefinition, SplitKeepSide, SplitMode, SplitRecipe, SplitTarget, SplitTool},
        work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference},
    },
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};

fn add_crossing_line_sketch(
    history: &mut DocumentEditHistory,
) -> (DocumentSketchId, SketchEntityId) {
    let token = history.begin("SplitLineSketch").unwrap();
    let (sketch_id, line_id) = history
        .edit(&token, |snapshot, ids| {
            let mut sketch = Sketch2D::default();
            let p0 = ids.allocate_sketch_entity().unwrap();
            let p1 = ids.allocate_sketch_entity().unwrap();
            let line = ids.allocate_sketch_entity().unwrap();
            sketch.add_point(p0, [0.5, -0.5]).unwrap();
            sketch.add_point(p1, [0.5, 1.5]).unwrap();
            sketch.add_line(line, [p0, p1]).unwrap();
            let id = ids.allocate_sketch().unwrap();
            snapshot
                .sketches
                .add(id, "SplitSketch", SketchPlane::xy(), sketch)
                .unwrap();
            solve_selected_sketches(snapshot);
            Ok::<_, oblikovati_model::history::HistoryError>((id, line))
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    (sketch_id, line_id)
}

fn add_midplane(history: &mut DocumentEditHistory) -> FeatureId {
    let token = history.begin("Midplane").unwrap();
    let plane_id = history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate_feature().unwrap();
            let def = WorkPlaneDefinition::Offset {
                base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
                distance: WorkPlaneDistance::Constant(0.5),
            };
            let reads = def.reads();
            let feature =
                oblikovati_model::feature::AuthoredFeature::new(id, "Work Plane1", def, reads);
            snapshot.work_geometry.append(feature).unwrap();
            Ok::<_, oblikovati_model::history::HistoryError>(id)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    plane_id
}

#[test]
fn split_all_faces_and_single_face_add_real_brep_edges_preserving_volume() {
    let mut history = owner(cube_document());
    let (sketch, entity) = add_crossing_line_sketch(&mut history);
    let all_faces = SplitDefinition::from_recipe(SplitRecipe {
        version: 1,
        mode: SplitMode::Faces,
        tool: SplitTool::SketchCurve { sketch, entity },
        target: SplitTarget::AllFaces {
            body: key("body", &[1]),
        },
        keep_side: SplitKeepSide::Both,
    })
    .unwrap();
    let token = history.begin("SplitAllFaces").unwrap();
    let split_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_split("Split1", all_faces, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        result
            .health()
            .iter()
            .all(|r| r.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        result.health()
    );
    let body = result.body(split_id).expect("expected split body");
    let summary = body
        .topology_summary(&history, document(), &key("body", &[1]))
        .unwrap();
    assert!(summary.solid);
    // Unit cube has 6 faces, 12 edges; cutting plane X=0.5 splits 4 faces (top, bottom, front, back) -> 10 faces.
    assert_eq!(
        summary.faces, 10,
        "expected 10 faces after All Faces split: {summary:?}"
    );
    assert!(summary.edges > 12, "expected new split edges: {summary:?}");
    let vol = body.volume(&history, document()).unwrap();
    assert!((vol - 1.0).abs() < 1e-6, "volume {vol}: expected 1.0");

    // Edit Split1 to restrict to only the top planar face ("end", &[1]).
    let single_face = SplitDefinition::from_recipe(SplitRecipe {
        version: 1,
        mode: SplitMode::Faces,
        tool: SplitTool::SketchCurve { sketch, entity },
        target: SplitTarget::Faces {
            body: key("body", &[1]),
            faces: vec![key("end", &[1])],
        },
        keep_side: SplitKeepSide::Both,
    })
    .unwrap();
    let token = history.begin("EditSplitSingleFace").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.edit_split(split_id, single_face)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let edited = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        edited
            .health()
            .iter()
            .all(|r| r.status == FeatureEvaluationStatus::Healthy),
        "edited health: {:?}",
        edited.health()
    );
    let body = edited
        .body(split_id)
        .expect("expected single-face split body");
    let summary = body
        .topology_summary(&history, document(), &key("body", &[1]))
        .unwrap();
    assert!(summary.solid);
    assert_eq!(
        summary.faces, 7,
        "expected 7 faces after splitting 1 face: {summary:?}"
    );
    assert!(
        summary.edges > 12,
        "expected new split edge on top face: {summary:?}"
    );
    let vol = body.volume(&history, document()).unwrap();
    assert!((vol - 1.0).abs() < 1e-6, "volume {vol}: expected 1.0");
}

#[test]
fn split_solid_produces_two_exact_halves_and_supports_single_side_edits() {
    let mut history = owner(cube_document());
    let plane_id = add_midplane(&mut history);
    let split_both = SplitDefinition::from_recipe(SplitRecipe {
        version: 1,
        mode: SplitMode::Solid,
        tool: SplitTool::WorkPlane(plane_id),
        target: SplitTarget::Solid {
            body: key("body", &[1]),
        },
        keep_side: SplitKeepSide::Both,
    })
    .unwrap();
    let token = history.begin("SplitSolidBoth").unwrap();
    let split_id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_split("Split1", split_both, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        result
            .health()
            .iter()
            .all(|r| r.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        result.health()
    );
    let bodies: Vec<_> = result.ordered_bodies().collect();
    assert_eq!(bodies.len(), 2, "expected 2 split solid bodies");
    let key0 = bodies[0].body_reference_key(&history, document()).unwrap();
    let key1 = bodies[1].body_reference_key(&history, document()).unwrap();
    assert_ne!(
        key0, key1,
        "expected distinct body reference keys for split halves"
    );
    let v0 = bodies[0].volume(&history, document()).unwrap();
    let v1 = bodies[1].volume(&history, document()).unwrap();
    assert!((v0 - 0.5).abs() < 1e-6, "half 0 volume {v0}: expected 0.5");
    assert!((v1 - 0.5).abs() < 1e-6, "half 1 volume {v1}: expected 0.5");

    // Edit to SideA and SideB: each keeps one exact 0.5 cm³ solid.
    for side in [SplitKeepSide::SideA, SplitKeepSide::SideB] {
        let single = SplitDefinition::from_recipe(SplitRecipe {
            version: 1,
            mode: SplitMode::Solid,
            tool: SplitTool::WorkPlane(plane_id),
            target: SplitTarget::Solid {
                body: key("body", &[1]),
            },
            keep_side: side,
        })
        .unwrap();
        let token = history.begin("EditSplitSide").unwrap();
        history
            .edit(&token, |snapshot, _| snapshot.edit_split(split_id, single))
            .unwrap();
        history.commit(&token, false).unwrap();

        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        let edited = history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap();
        assert!(
            edited
                .health()
                .iter()
                .all(|r| r.status == FeatureEvaluationStatus::Healthy),
            "side {side:?} health: {:?}",
            edited.health()
        );
        let bodies: Vec<_> = edited.ordered_bodies().collect();
        assert_eq!(bodies.len(), 1, "expected 1 solid for {side:?}");
        let v = bodies[0].volume(&history, document()).unwrap();
        assert!(
            (v - 0.5).abs() < 1e-6,
            "side {side:?} volume {v}: expected 0.5"
        );
    }
}
