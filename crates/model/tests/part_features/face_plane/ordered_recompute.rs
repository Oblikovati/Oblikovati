// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn ordered_part_replay_solves_face_sketch_after_its_source_plane() {
    use oblikovati_model::{
        document_sketch::{PlacedSketchHealth, SketchEvaluation, SketchPlacement},
        history::DocumentSketchCreation,
    };
    let snapshot = face_document();
    let datums = snapshot
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap()
        .datum_evaluation();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let request = DocumentSketchCreation {
        name: "Face sketch".into(),
        placement: SketchPlacement::WorkPlane(FeatureId(3)),
        orientation: None,
        project_origin: true,
    };
    history
        .preview(|candidate, ids| {
            let sketch = candidate.create_document_sketch_with_datums(&request, ids, &datums)?;
            candidate.sketches = candidate
                .solve_pending_document_sketches(Default::default(), || false)
                .candidate;
            let (replay, sketches) = candidate
                .recompute_part_with_sketches(&mut FaceEvaluator::default(), Default::default())
                .unwrap();
            let mut updated = candidate.clone();
            updated.sketches = sketches;
            let current = replay.datum_evaluation();
            assert_eq!(
                updated
                    .resolve_sketch_plane_with_datums(sketch, &current)
                    .unwrap()
                    .recipe()
                    .origin,
                [1.5, 0., 0.]
            );
            let health = updated
                .document_sketch_health_with_datums(sketch, &current)
                .unwrap();
            assert!(
                matches!(
                    health,
                    PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
                ),
                "{health:?}"
            );
            Ok::<_, oblikovati_model::history::DocumentSketchCreationError>(())
        })
        .unwrap();
}

#[test]
fn ordered_recompute_builds_a_feature_from_a_face_hosted_sketch() {
    verify_face_hosted_prism(false);
    verify_face_hosted_prism(true);
}
fn verify_face_hosted_prism(direct: bool) {
    use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
    let snapshot = face_document();
    let part = snapshot
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap();
    let face = WorkPlaneFaceReference::new(face_key(1)).unwrap();
    let datums = if direct {
        part.datum_evaluation_for_face(&FaceEvaluator::default(), &face)
            .unwrap()
    } else {
        part.datum_evaluation()
    };
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let request = DocumentSketchCreation {
        name: "Face profile".into(),
        placement: if direct {
            SketchPlacement::Face(face)
        } else {
            SketchPlacement::WorkPlane(FeatureId(3))
        },
        orientation: None,
        project_origin: false,
    };
    history
        .preview(|candidate, ids| {
            let sketch = candidate.create_document_sketch_with_datums(&request, ids, &datums)?;
            use oblikovati_model::sketch::{PlanarConstraint, Sketch2D};
            let mut geometry = Sketch2D::default();
            let positions = [[0., 0.], [1., 0.], [1., 1.], [0., 1.]];
            let points: Vec<_> = (0..4)
                .map(|_| ids.allocate_sketch_entity().unwrap())
                .collect();
            for (index, position) in positions.into_iter().enumerate() {
                let point = points[index];
                geometry.add_point(point, position).unwrap();
                geometry
                    .add_constraint(
                        ids.allocate_sketch_constraint(sketch).unwrap(),
                        PlanarConstraint::Fix { point, position },
                    )
                    .unwrap();
            }
            let mut lines = Vec::new();
            for index in 0..4 {
                let endpoints = [points[index], points[(index + 1) % 4]];
                let line = ids.allocate_sketch_entity().unwrap();
                geometry.add_line(line, endpoints).unwrap();
                lines.push(
                    oblikovati_model::feature::sketch_prism::SketchDirectedLine {
                        line,
                        reversed: false,
                    },
                );
            }
            candidate
                .sketches
                .replace_geometry(sketch, geometry)
                .unwrap();
            let prism = SketchPrismDefinition::new(
                sketch,
                oblikovati_model::feature::sketch_prism::SketchPrismSelection {
                    outer: lines,
                    holes: Vec::new(),
                },
                [
                    DatumLength::Constant(0.),
                    DatumLength::Constant(0.),
                    DatumLength::Constant(1.),
                ],
            )
            .unwrap();
            let feature_id = ids.allocate_feature().unwrap();
            assert_eq!(feature_id, FeatureId(4));
            let feature =
                AuthoredFeature::sketch_prism(feature_id, "Dependent prism", prism, candidate)
                    .unwrap();
            candidate.work_geometry.append(feature).unwrap();
            let before = candidate
                .evaluate_part_features(&mut FaceEvaluator::default())
                .unwrap();
            assert!(before.body(FeatureId(4)).is_none());
            let (result, _) = candidate
                .recompute_part_with_sketches(&mut FaceEvaluator::default(), Default::default())
                .unwrap();
            assert!(result.body(FeatureId(4)).is_some());
            let (failed, _) = candidate
                .recompute_part_with_sketches(
                    &mut FaceEvaluator {
                        fail_body: true,
                        fail_face: false,
                    },
                    Default::default(),
                )
                .unwrap();
            assert!(failed.body(FeatureId(4)).is_none());
            assert!(matches!(
                &failed.health()[3].status,
                FeatureEvaluationStatus::Blocked { .. }
            ));
            Ok::<_, oblikovati_model::history::DocumentSketchCreationError>(())
        })
        .unwrap();
}

#[test]
fn ordered_recompute_discards_cancelled_sketch_work() {
    use oblikovati_model::feature::part::PartFeatureProgramError;
    let snapshot = face_document();
    let before = snapshot.work_geometry.part_recipe().unwrap();
    let mut calls = 0;
    let result = snapshot.recompute_part_with_sketches_cancellable(
        &mut FaceEvaluator::default(),
        Default::default(),
        || {
            calls += 1;
            calls == 3
        },
    );
    assert!(matches!(
        result,
        Err(PartFeatureProgramError::Cancelled { .. })
    ));
    assert_eq!(snapshot.work_geometry.part_recipe().unwrap(), before);
}
