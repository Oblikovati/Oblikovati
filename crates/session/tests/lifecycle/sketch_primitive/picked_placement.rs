// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::SketchPlacementFieldLock,
    sketch::{SketchPlacementBinding, SketchPlacementRecipe, SketchPlacementRelation},
};

fn midpoint_recipe() -> SketchPlacementRecipe {
    let (sin, cos) = 30_f64.to_radians().sin_cos();
    let mut recipe = SketchPlacementRecipe::line_chain(
        &[[-0.5 * cos, -0.5 * sin], [0.5 * cos, 0.5 * sin]],
        false,
    )
    .unwrap();
    recipe.points.push([0., 0.]);
    recipe.picked_points.insert(2, SketchEntityId(1));
    recipe
        .relations
        .push(SketchPlacementRelation::Midpoint { point: 2, line: 0 });
    recipe.fields[1].binding = SketchPlacementBinding::LineAngle(0);
    recipe
}

fn midpoint_locks() -> [Option<SketchPlacementFieldLock>; 2] {
    ["10 mm", "30 deg"].map(|expression| {
        Some(SketchPlacementFieldLock {
            reference: false,
            expression: expression.into(),
            text_point: None,
        })
    })
}

#[test]
fn picked_origin_midpoint_recipe_keeps_zero_degrees_of_freedom() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let report = session
        .create_sketch_placement(sketch, &midpoint_recipe(), &midpoint_locks())
        .unwrap();
    assert_eq!(
        session.analyze_sketch(sketch).unwrap().degrees_of_freedom,
        0
    );
    assert_eq!(report.points[2], SketchEntityId(1));
}

#[test]
fn picked_origin_preview_commit_and_history_agree() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let before = session.snapshot().clone();
    let verifier = super::super::placement_verification::RecordingCircularVerifier::default();
    let preview = session
        .preview_sketch_placement(sketch, &midpoint_recipe(), &midpoint_locks(), &verifier)
        .unwrap();
    let report = session
        .create_sketch_placement(sketch, &midpoint_recipe(), &midpoint_locks())
        .unwrap();
    assert!(preview.geometry.same_recipe(geometry(&session, sketch)));
    assert_eq!(report.points[2], SketchEntityId(1));
    let after = session.snapshot().clone();
    session.undo().unwrap();
    assert!(before.sketches.same_recipe(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(after.sketches.same_recipe(&session.snapshot().sketches));
    assert_eq!(
        session.analyze_sketch(sketch).unwrap().degrees_of_freedom,
        0
    );
}

#[test]
fn stale_and_invalid_point_picks_are_rejected_atomically() {
    for (index, id, position) in [(2, 999, [0., 0.]), (9, 1, [0., 0.]), (2, 1, [1., 0.])] {
        let mut session = session(None);
        let sketch = empty_sketch(&mut session, true);
        let before = session.snapshot().clone();
        let mut recipe = midpoint_recipe();
        recipe.picked_points.clear();
        recipe.picked_points.insert(index, SketchEntityId(id));
        recipe.points[2] = position;
        assert!(
            session
                .create_sketch_placement(sketch, &recipe, &midpoint_locks())
                .is_err()
        );
        assert!(before.sketches.same_recipe(&session.snapshot().sketches));
    }
}

fn set_inference(
    session: &mut DocumentSession<RecordingObserver>,
    infer: bool,
    apply: bool,
    persist: bool,
) {
    session.begin("Set inference").unwrap();
    session
        .edit(|snapshot, _| {
            let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
            settings.infer_constraints = infer;
            settings.auto_apply_constraints = apply;
            settings.persist_inferred_constraints = persist;
            snapshot.sketch_settings = Some(settings);
            Ok::<_, oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    session.end().unwrap();
}

#[test]
fn explicit_point_pick_is_not_automatic_inference() {
    for (infer, apply, persist) in [
        (false, false, false),
        (true, false, true),
        (true, true, false),
        (true, true, true),
    ] {
        let mut session = session(None);
        let sketch = empty_sketch(&mut session, true);
        set_inference(&mut session, infer, apply, persist);
        let report = session
            .create_sketch_placement(sketch, &midpoint_recipe(), &midpoint_locks())
            .unwrap();
        assert_eq!(report.points[2], SketchEntityId(1));
        assert_eq!(
            session.analyze_sketch(sketch).unwrap().degrees_of_freedom,
            0
        );
    }
}

#[test]
fn placement_preserves_the_existing_geometry_of_a_picked_point() {
    let mut source = Sketch2D::default();
    source.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    source.add_point(SketchEntityId(2), [2., 0.]).unwrap();
    source
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    let mut session = super::super::sketch_fixture::owned_sketch_session(None, "Source", source);
    let sketch = DocumentSketchId(1);
    set_inference(&mut session, false, false, false);
    session
        .create_sketch_placement(sketch, &midpoint_recipe(), &midpoint_locks())
        .unwrap();
    assert_eq!(
        geometry(&session, sketch)
            .point(SketchEntityId(1))
            .unwrap()
            .position,
        [0., 0.]
    );
    assert_eq!(
        geometry(&session, sketch)
            .point(SketchEntityId(2))
            .unwrap()
            .position,
        [2., 0.]
    );
}

#[test]
fn picked_midpoint_remains_on_origin_after_dimension_edit() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let report = session
        .create_sketch_placement(sketch, &midpoint_recipe(), &midpoint_locks())
        .unwrap();
    let edit = oblikovati_model::history::SketchDimensionEdit {
        target_expression: Some("12 mm".into()),
        ..Default::default()
    };
    session
        .edit_sketch_dimension(sketch, report.dimensions[0].1.constraint, &edit)
        .unwrap();
    assert_eq!(
        geometry(&session, sketch)
            .point(SketchEntityId(1))
            .unwrap()
            .position,
        [0., 0.]
    );
    assert_eq!(
        session.analyze_sketch(sketch).unwrap().degrees_of_freedom,
        0
    );
    session.undo().unwrap();
    session.redo().unwrap();
    assert_eq!(
        geometry(&session, sketch)
            .point(SketchEntityId(1))
            .unwrap()
            .position,
        [0., 0.]
    );
}

#[test]
fn unselected_midpoint_does_not_acquire_a_coincident_origin() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let mut recipe = midpoint_recipe();
    recipe.picked_points.clear();
    let report = session
        .create_sketch_placement(sketch, &recipe, &midpoint_locks())
        .unwrap();
    assert_ne!(report.points[2], SketchEntityId(1));
    assert_eq!(
        session.analyze_sketch(sketch).unwrap().degrees_of_freedom,
        2
    );
}
