// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchPoint2D},
};
use oblikovati_solve::nonlinear::{PreferenceTermination, SolveOptions};

fn points() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    sketch.add_point(SketchEntityId(2), [8.0, 4.0]).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Coincident {
                points: [SketchEntityId(1), SketchEntityId(2)],
            },
        )
        .unwrap();
    sketch
}

#[test]
fn preferred_first_point_is_preserved_without_reducing_sketch_freedom() {
    let sketch = points();
    let preference = SketchPoint2D {
        id: SketchEntityId(1),
        position: [0.0, 0.0],
    };
    let (outcome, status) = sketch
        .solve_with_point_preferences(
            &ParameterCollection::new(Default::default()),
            &[],
            &[preference],
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    assert_eq!(status, PreferenceTermination::Converged);
    for id in [SketchEntityId(1), SketchEntityId(2)] {
        let point = outcome.candidate().point(id).unwrap();
        assert!(
            point
                .position
                .iter()
                .all(|coordinate| coordinate.abs() < 1e-7)
        );
    }
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 2);
}

#[test]
fn preferred_point_conflicting_with_grounded_source_is_unavailable() {
    let sketch = points();
    let ground = SketchPoint2D {
        id: SketchEntityId(2),
        position: [8.0, 4.0],
    };
    let preference = SketchPoint2D {
        id: SketchEntityId(1),
        position: [0.0, 0.0],
    };
    let (outcome, status) = sketch
        .solve_with_point_preferences(
            &ParameterCollection::new(Default::default()),
            &[ground],
            &[preference],
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    assert_eq!(status, PreferenceTermination::Unavailable);
    assert_eq!(
        outcome
            .candidate()
            .point(SketchEntityId(2))
            .unwrap()
            .position,
        [8.0, 4.0]
    );
}

#[test]
fn grounded_preference_and_invalid_targets_are_distinguished() {
    let sketch = points();
    let point = SketchPoint2D {
        id: SketchEntityId(2),
        position: [8.0, 4.0],
    };
    let parameters = ParameterCollection::new(Default::default());
    let (_, status) = sketch
        .solve_with_point_preferences(
            &parameters,
            &[point],
            &[point],
            SolveOptions::default(),
            || false,
        )
        .unwrap();
    assert_eq!(status, PreferenceTermination::Converged);
    for targets in [
        vec![point, point],
        vec![SketchPoint2D {
            id: SketchEntityId(99),
            position: [0.0, 0.0],
        }],
        vec![SketchPoint2D {
            id: point.id,
            position: [f64::NAN, 0.0],
        }],
    ] {
        assert!(
            sketch
                .solve_with_point_preferences(
                    &parameters,
                    &[],
                    &targets,
                    SolveOptions::default(),
                    || false
                )
                .is_err()
        );
    }
}

#[test]
fn cancellation_is_not_hidden_by_a_conflicting_grounded_preference() {
    let sketch = points();
    let anchor = SketchPoint2D {
        id: SketchEntityId(2),
        position: [8.0, 4.0],
    };
    let target = SketchPoint2D {
        id: anchor.id,
        position: [0.0, 0.0],
    };
    let (_, status) = sketch
        .solve_with_point_preferences(
            &ParameterCollection::new(Default::default()),
            &[anchor],
            &[target],
            SolveOptions::default(),
            || true,
        )
        .unwrap();
    assert_eq!(status, PreferenceTermination::Cancelled);
}
