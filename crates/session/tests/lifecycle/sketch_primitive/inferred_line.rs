// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::SketchLineCreation;
fn request() -> SketchLineCreation {
    SketchLineCreation {
        endpoints: [[0.0001, 0.0001], [4.0, 0.01]],
        construction: false,
        centerline: true,
    }
}
#[test]
fn inferred_line_joins_an_open_group_and_recomputes_before_publication() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    session
        .set_sketch_inference_scope(sketch, INFERENCE_SCOPE_ALL_GEOMETRY, &[])
        .unwrap();
    session.begin("Author geometry").unwrap();
    let report = session
        .create_inferred_sketch_line(sketch, &request())
        .unwrap();
    assert_eq!(report.point_constraints.len(), 1);
    assert_eq!(report.direction.unwrap().geometric_index, 1);
    assert_numeric_verification(&session, sketch, 1);
    assert_centerline(&session, sketch, report.primitive.entity);
    session.end().unwrap();
    let label = session.undo_labels().last().copied().unwrap();
    assert_eq!(label, "Author geometry");
    session.undo().unwrap();
    assert!(geometry(&session, sketch).lines().is_empty());
    session.redo().unwrap();
    assert_numeric_verification(&session, sketch, 1);
}
#[test]
fn rejected_reply_discards_line_and_relations_but_reserves_all_their_ids() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    session
        .set_sketch_inference_scope(sketch, INFERENCE_SCOPE_ALL_GEOMETRY, &[])
        .unwrap();
    let before = session.snapshot().sketches.clone();
    let result = session.create_inferred_sketch_line_projected(sketch, &request(), |result| {
        let report = result.unwrap();
        assert_eq!(report.primitive.entity, SketchEntityId(4));
        assert_eq!(report.direction.unwrap().constraint.0, 2);
        Err::<(), _>("reply refused")
    });
    assert!(result.unwrap_err().to_string().contains("reply refused"));
    assert_eq!(session.snapshot().sketches, before);
    let report = session
        .create_inferred_sketch_line(sketch, &request())
        .unwrap();
    assert_eq!(report.primitive.entity, SketchEntityId(7));
    assert_eq!(report.point_constraints[0].0, 3);
    assert_eq!(report.direction.unwrap().constraint.0, 4);
}
#[test]
fn commit_veto_keeps_no_geometry_and_the_next_attempt_uses_new_ids() {
    let mut session = prepared_session(Some(EVENT_TRANSACTION_COMMITTED), 0);
    let sketch = DocumentSketchId(1);
    assert!(
        session
            .create_inferred_sketch_line(sketch, &request())
            .is_err()
    );
    assert!(geometry(&session, sketch).entity_ids().is_empty());
    assert_next_vetoed_ids(&mut session, sketch);
    assert!(geometry(&session, sketch).entity_ids().is_empty());
    assert!(session.undo_labels().is_empty());
}
#[test]
fn document_working_scale_sets_the_snap_reach() {
    for (scale, expected_snaps) in [(0.1, 1), (10.0, 0)] {
        let mut session = scaled_session(scale);
        session
            .create_sketch_primitive(DocumentSketchId(1), &point([0.0, 0.0]))
            .unwrap();
        session
            .set_sketch_inference_scope(DocumentSketchId(1), INFERENCE_SCOPE_ALL_GEOMETRY, &[])
            .unwrap();
        let mut input = request();
        input.endpoints[0] = [0.0005, 0.0005];
        let report = session
            .create_inferred_sketch_line(DocumentSketchId(1), &input)
            .unwrap();
        assert_eq!(report.point_constraints.len(), expected_snaps);
    }
}

fn assert_centerline(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    line: SketchEntityId,
) {
    assert!(geometry(session, sketch).is_centerline(line).unwrap());
}

fn assert_next_vetoed_ids(
    session: &mut DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
) {
    assert!(
        session
            .create_inferred_sketch_line_projected(sketch, &request(), |result| {
                let report = result.unwrap();
                assert_eq!(report.primitive.entity, SketchEntityId(6));
                assert_eq!(report.direction.unwrap().constraint.0, 2);
                Ok::<_, String>(report)
            })
            .is_err()
    );
}

fn scaled_session(scale: f64) -> DocumentSession<RecordingObserver> {
    let mut snapshot = prepared_snapshot();
    snapshot.parameters =
        ParameterCollection::new(UnitsOfMeasure::default().with_working_scale(scale).unwrap());
    snapshot
        .parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    DocumentSession::new(17, history, RecordingObserver::default())
}

#[test]
fn nonpersistent_direction_aligns_line_and_stays_unsaved_after_undo_redo() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    session.begin("Inference settings").unwrap();
    session
        .edit(|snapshot, _| {
            let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
            settings.persist_inferred_constraints = false;
            snapshot.sketch_settings = Some(settings);
            Ok::<_, oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    session.end().unwrap();
    let before = session.snapshot().sketches.clone();
    let input = SketchLineCreation {
        endpoints: [[10.0, 10.0], [14.0, 10.01]],
        construction: false,
        centerline: false,
    };
    let report = session.create_inferred_sketch_line(sketch, &input).unwrap();
    assert!(report.inference.direction.is_some());
    assert!(
        report.direction.is_none(),
        "temporary direction must not be saved"
    );
    let sketch_geometry = geometry(&session, sketch);
    let first = sketch_geometry
        .point(report.primitive.points[0])
        .unwrap()
        .position;
    let second = sketch_geometry
        .point(report.primitive.points[1])
        .unwrap()
        .position;
    assert!((first[1] - second[1]).abs() < 1e-9);
    let after = session.snapshot().sketches.clone();
    session.undo().unwrap();
    assert!(before.same_recipe(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(after.same_evaluated_state(&session.snapshot().sketches));
}

#[test]
fn both_persistence_modes_keep_parallel_reference_geometry_fixed() {
    for persist in [false, true] {
        let mut session = session(None);
        let sketch = empty_sketch(&mut session, false);
        let source = SketchPrimitiveCreation::Line {
            endpoints: [[2.0, 4.0], [6.0, 5.0]].map(SketchPointInput::New),
            construction: false,
            centerline: false,
        };
        session.create_sketch_primitive(sketch, &source).unwrap();
        session.begin("Inference settings").unwrap();
        session
            .edit(|snapshot, _| {
                let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
                settings.persist_inferred_constraints = persist;
                settings.constraint_priority = PRIORITY_PARALLEL_PERPENDICULAR;
                snapshot.sketch_settings = Some(settings);
                Ok::<_, oblikovati_model::parameter::ParameterError<u64>>(())
            })
            .unwrap();
        session.end().unwrap();
        session
            .set_sketch_inference_scope(sketch, INFERENCE_SCOPE_ALL_GEOMETRY, &[])
            .unwrap();
        let before = geometry(&session, sketch).clone();
        let request = SketchLineCreation {
            endpoints: [[10.0, 10.0], [14.0, 11.01]],
            construction: false,
            centerline: false,
        };
        let report = session
            .create_inferred_sketch_line(sketch, &request)
            .unwrap();
        assert_eq!(report.direction.is_some(), persist);
        assert_eq!(report.inference.direction.unwrap().kind, INFER_PARALLEL);
        let after = geometry(&session, sketch);
        for source_point in before.points() {
            assert_eq!(
                after.point(source_point.id).unwrap().position,
                source_point.position
            );
        }
        let line = &report.primitive.points;
        let a = after.point(line[0]).unwrap().position;
        let b = after.point(line[1]).unwrap().position;
        assert!(((b[0] - a[0]) - 4.0 * (b[1] - a[1])).abs() < 1e-7);
    }
}

#[test]
fn disabled_inference_or_auto_apply_does_not_align_or_save_direction() {
    for (infer, auto) in [(false, true), (true, false)] {
        let mut session = session(None);
        let sketch = empty_sketch(&mut session, false);
        session.begin("Inference settings").unwrap();
        session
            .edit(|snapshot, _| {
                let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
                settings.infer_constraints = infer;
                settings.auto_apply_constraints = auto;
                settings.persist_inferred_constraints = false;
                snapshot.sketch_settings = Some(settings);
                Ok::<_, oblikovati_model::parameter::ParameterError<u64>>(())
            })
            .unwrap();
        session.end().unwrap();
        let input = SketchLineCreation {
            endpoints: [[10.0, 10.0], [14.0, 10.01]],
            construction: false,
            centerline: false,
        };
        let report = session.create_inferred_sketch_line(sketch, &input).unwrap();
        assert!(report.direction.is_none());
        assert_eq!(report.inference.direction.is_some(), infer);
        let placed = geometry(&session, sketch);
        assert_eq!(
            placed.point(report.primitive.points[0]).unwrap().position,
            input.endpoints[0]
        );
        assert_eq!(
            placed.point(report.primitive.points[1]).unwrap().position,
            input.endpoints[1]
        );
    }
}

#[test]
#[ignore = "measure large-sketch line creation in release mode"]
fn measure_inferred_line_on_large_sketch() {
    use std::time::Instant;
    for count in [200_u64, 1000] {
        let source = large_line_source(count);
        let mut samples = Vec::new();
        for iteration in 0..13 {
            let mut session = super::super::sketch_fixture::owned_sketch_session(
                None,
                "Benchmark",
                source.clone(),
            );
            let sketch = DocumentSketchId(1);
            session
                .set_sketch_inference_scope(sketch, INFERENCE_SCOPE_ALL_GEOMETRY, &[])
                .unwrap();
            let input = SketchLineCreation {
                endpoints: [[10.0, 10.0], [14.0, 10.01]],
                construction: false,
                centerline: false,
            };
            let started = Instant::now();
            let report = session.create_inferred_sketch_line(sketch, &input).unwrap();
            let elapsed = started.elapsed();
            assert!(report.direction.is_some());
            let geometry = geometry(&session, sketch);
            let a = geometry.point(report.primitive.points[0]).unwrap().position;
            let b = geometry.point(report.primitive.points[1]).unwrap().position;
            assert!((a[1] - b[1]).abs() < 1e-8);
            assert_eq!(
                geometry.point(SketchEntityId(1)).unwrap().position,
                [1000.0, 100.0]
            );
            if iteration >= 3 {
                samples.push(elapsed);
            }
        }
        samples.sort();
        println!(
            "line_creation lines={count} n={} min_ms={:.3} median_ms={:.3} max_ms={:.3}",
            samples.len(),
            samples[0].as_secs_f64() * 1000.0,
            samples[5].as_secs_f64() * 1000.0,
            samples[9].as_secs_f64() * 1000.0
        );
    }
}
fn large_line_source(count: u64) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    for index in 0..count {
        let a = SketchEntityId(3 * index + 1);
        let b = SketchEntityId(3 * index + 2);
        geometry
            .add_point(a, [1000.0 + index as f64 * 3.0, 100.0])
            .unwrap();
        geometry
            .add_point(b, [1001.0 + index as f64 * 3.0, 100.0])
            .unwrap();
        geometry
            .add_line(SketchEntityId(3 * index + 3), [a, b])
            .unwrap();
    }
    geometry
}
