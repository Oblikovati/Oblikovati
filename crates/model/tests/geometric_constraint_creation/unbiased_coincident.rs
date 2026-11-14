// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_solve::nonlinear::PreferenceTermination;

#[test]
fn coincident_creation_without_an_edit_preference_splits_motion() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_point(entity(2), [8.0, 4.0]).unwrap();
    let mut history = owner(sketch);
    create(
        &mut history,
        &direct(PlanarConstraint::Coincident {
            points: [entity(1), entity(2)],
        }),
    );
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    for id in [entity(1), entity(2)] {
        let position = geometry.point(id).unwrap().position;
        assert!((position[0] - 4.0).abs() < 1e-6);
        assert!((position[1] - 2.0).abs() < 1e-6);
    }
}

#[test]
fn explicit_point_preference_is_an_atomic_history_step() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_point(entity(2), [8.0, 4.0]).unwrap();
    let mut history = owner(sketch);
    let token = history.begin("Coincident with preference").unwrap();
    let result = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_geometric_constraint_preferred(
                DocumentSketchId(1),
                &direct(PlanarConstraint::Coincident {
                    points: [entity(1), entity(2)],
                }),
                entity(1),
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(result.preference, Some(PreferenceTermination::Converged));
    assert_eq!(result.degrees_of_freedom, 2);
    check_second_point(&history, [0.0, 0.0]);
    history.undo().unwrap();
    check_second_point(&history, [8.0, 4.0]);
    history.redo().unwrap();
    check_second_point(&history, [0.0, 0.0]);
}

fn check_second_point(history: &DocumentEditHistory, expected: [f64; 2]) {
    let sketch = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap();
    for (actual, expected) in sketch
        .geometry()
        .point(entity(2))
        .unwrap()
        .position
        .into_iter()
        .zip(expected)
    {
        assert!(
            (actual - expected).abs() < 1e-7,
            "actual {actual}, expected {expected}"
        );
    }
}

#[test]
fn fixed_second_point_reports_unavailable_preference_without_losing_the_relation() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_point(entity(2), [8.0, 4.0]).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Fix {
                point: entity(2),
                position: [8.0, 4.0],
            },
        )
        .unwrap();
    let mut history = owner(sketch);
    let token = history.begin("Coincident with fixed point").unwrap();
    let result = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_geometric_constraint_preferred(
                DocumentSketchId(1),
                &direct(PlanarConstraint::Coincident {
                    points: [entity(1), entity(2)],
                }),
                entity(1),
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(result.preference, Some(PreferenceTermination::Unavailable));
    let sketch = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap();
    assert_eq!(
        sketch.geometry().point(entity(1)).unwrap().position,
        [8.0, 4.0]
    );
    assert_eq!(sketch.geometry().geometric_constraints().count(), 2);
}
