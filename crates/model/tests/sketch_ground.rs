// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{parameter::ParameterCollection, sketch::*};
fn id(value: u64) -> SketchEntityId {
    SketchEntityId(value)
}
fn fixture(scale: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (point, position) in [(1, [2.0, -3.0]), (2, [4.0, -3.0]), (3, [2.0, -1.0])] {
        sketch
            .add_point(id(point), position.map(|value| value * scale))
            .unwrap();
    }
    sketch.add_line(id(4), [id(2), id(3)]).unwrap();
    sketch.add_circle(id(5), id(1), 2.0 * scale).unwrap();
    sketch.add_arc(id(6), id(1), [id(2), id(3)], true).unwrap();
    sketch
        .add_ellipse(SketchEllipse2D::new(id(7), id(1), [3.0, 4.0], [3.0 * scale, scale]).unwrap())
        .unwrap();
    sketch
}
fn anchors(sketch: &Sketch2D, points: &[u64]) -> Vec<SketchPoint2D> {
    points
        .iter()
        .map(|point| *sketch.point(id(*point)).unwrap())
        .collect()
}
fn grounded(scale: f64, points: &[u64]) -> Sketch2D {
    let mut sketch = fixture(scale);
    let points = anchors(&sketch, points);
    sketch
        .add_constraint(SketchConstraintId(1), PlanarConstraint::Ground { points })
        .unwrap();
    sketch
}
#[test]
fn ground_coordinates_restore_all_native_entity_points_at_three_scales() {
    for scale in [1e-6, 1.0, 1e6] {
        for (entity, points, dof) in [
            (3, vec![3], 7),
            (4, vec![2, 3], 5),
            (5, vec![1], 7),
            (6, vec![1, 2, 3], 4),
            (7, vec![1], 7),
        ] {
            assert_grounded_solve(scale, entity, &points, dof);
        }
    }
}
fn assert_grounded_solve(scale: f64, entity: u64, points: &[u64], dof: usize) {
    let mut sketch = grounded(scale, points);
    let expected = anchors(&sketch, points);
    sketch
        .translate_entities(&[id(entity)], [0.25 * scale, -0.5 * scale])
        .unwrap();
    let parameters = ParameterCollection::new(Default::default());
    let solved = sketch
        .solve(&parameters, ground_options(scale), || false)
        .unwrap();
    assert_eq!(
        solved.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
    assert_eq!(solved.numerical().structure.degrees_of_freedom, dof);
    assert_restored_points(solved.candidate(), &expected, scale);
}
fn ground_options(scale: f64) -> oblikovati_solve::nonlinear::SolveOptions {
    // The solver's coordinate scale also includes the ellipse angle. Request the
    // same coordinate accuracy at small lengths instead of assuming a physical-unit policy.
    oblikovati_solve::nonlinear::SolveOptions {
        relative_tolerance: oblikovati_solve::nonlinear::SolveOptions::default().relative_tolerance
            * scale.min(1.0),
        ..Default::default()
    }
}
fn assert_restored_points(candidate: &Sketch2D, expected: &[SketchPoint2D], scale: f64) {
    for point in expected {
        let actual = candidate.point(point.id).unwrap().position;
        assert!(
            actual
                .into_iter()
                .zip(point.position)
                .all(|(a, b)| (a - b).abs() < 1e-8 * scale)
        );
    }
    assert_eq!(
        candidate.ellipse(id(7)).unwrap().radii(),
        [3.0 * scale, scale]
    );
}
#[test]
fn ground_derivatives_are_exact_and_only_point_coordinates_are_fixed() {
    let sketch = grounded(1.0, &[1, 2, 3]);
    let parameters = ParameterCollection::new(Default::default());
    let solved = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    for (row, derivative) in solved.numerical().evaluation.jacobian()[..6]
        .iter()
        .enumerate()
    {
        let mut expected = vec![0.0; 10];
        expected[row] = 1.0;
        assert_eq!(derivative, &expected);
    }
    let rows = sketch
        .inspect_entities(&parameters, &[], Default::default())
        .unwrap();
    assert_eq!(rows[0].movement, oblikovati_api::types::MOVEABLE_FIXED);
    assert_eq!(rows[6].movement, oblikovati_api::types::MOVEABLE_FREE);
}
#[test]
fn ground_schema_preserves_anchor_bits_and_refuses_legacy_versions() {
    let sketch = grounded(1.0, &[1, 2]);
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 8);
    let bytes = serde_json::to_vec(&recipe).unwrap();
    assert!(
        sketch
            .same_recipe(&Sketch2D::from_recipe(&serde_json::from_slice(&bytes).unwrap()).unwrap())
    );
    assert_legacy_ground_declines(&recipe);
    assert_anchor_bits();
}
fn assert_legacy_ground_declines(recipe: &SketchRecipe) {
    for version in 1..=7 {
        let mut legacy = recipe.clone();
        legacy.version = version;
        assert!(
            Sketch2D::from_recipe(&legacy)
                .unwrap_err()
                .to_string()
                .contains("expected version 8")
        );
    }
}
fn assert_anchor_bits() {
    let mut positive = fixture(1.0);
    let point = SketchPoint2D {
        id: id(1),
        position: [0.0, 0.0],
    };
    positive
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Ground {
                points: vec![point],
            },
        )
        .unwrap();
    let mut recipe = positive.recipe();
    if let PlanarConstraint::Ground { points } = &mut recipe.constraints[0].relation {
        points[0].position[1] = -0.0;
    }
    assert!(!positive.same_recipe(&Sketch2D::from_recipe(&recipe).unwrap()));
}
#[test]
fn malformed_ground_anchors_are_refused_without_owner_changes() {
    let mut sketch = fixture(1.0);
    let before = sketch.clone();
    let point = *sketch.point(id(1)).unwrap();
    for points in invalid_anchor_sets(point) {
        assert!(
            sketch
                .add_constraint(SketchConstraintId(1), PlanarConstraint::Ground { points })
                .is_err()
        );
        assert!(sketch.same_recipe(&before));
    }
}
fn invalid_anchor_sets(point: SketchPoint2D) -> Vec<Vec<SketchPoint2D>> {
    vec![
        vec![],
        vec![point, point],
        vec![SketchPoint2D {
            id: id(99),
            ..point
        }],
        vec![SketchPoint2D {
            position: [f64::NAN, 0.0],
            ..point
        }],
    ]
}

#[test]
fn incompatible_frozen_anchor_sets_report_their_authored_constraints() {
    let mut sketch = grounded(1.0, &[1]);
    let mut points = anchors(&sketch, &[1]);
    points[0].position[0] += 2.0;
    sketch
        .add_constraint(SketchConstraintId(2), PlanarConstraint::Ground { points })
        .unwrap();
    let parameters = ParameterCollection::new(Default::default());
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_ne!(
        outcome.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
    let conflicts = outcome.conflicting_constraints();
    assert_eq!(conflicts.len(), 2);
    assert!(conflicts.contains(&SketchConstraintId(1)));
    assert!(conflicts.contains(&SketchConstraintId(2)));
}
