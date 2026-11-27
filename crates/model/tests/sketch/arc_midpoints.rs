// SPDX-License-Identifier: GPL-2.0-only
use super::arcs::driven_arc;
use super::*;
use std::f64::consts::FRAC_1_SQRT_2;
fn midpoint_sketch(counter_clockwise: bool, scale: f64) -> Sketch2D {
    let mut sketch = driven_arc(scale, counter_clockwise);
    sketch
        .add_point(
            entity(5),
            [if counter_clockwise {
                -3.0 * scale
            } else {
                3.0 * scale
            }; 2],
        )
        .unwrap();
    relation(
        &mut sketch,
        5,
        PlanarConstraint::ArcMidpoint {
            point: entity(5),
            arc: entity(4),
        },
    );
    sketch
}
#[test]
fn both_sweeps_solve_from_wrong_side_and_follow_radius_and_endpoint_edits_with_zero_dof() {
    for counter_clockwise in [false, true] {
        let original = midpoint_sketch(counter_clockwise, 1.0);
        let mut sketch = original.clone();
        let mut parameters = parameters();
        for radius in [5.0, 8.0] {
            parameters
                .set_expression(1, &format!("{radius} cm"))
                .unwrap();
            let result = sketch
                .solve(&parameters, SolveOptions::default(), || false)
                .unwrap();
            assert_eq!(result.numerical().termination, SolveTermination::Converged);
            assert_eq!(result.numerical().structure.rank, 8);
            assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
            let component = radius * FRAC_1_SQRT_2 * if counter_clockwise { 1.0 } else { -1.0 };
            assert_point(result.candidate(), 5, [component, component]);
            assert_eq!(result.candidate().constraints(), original.constraints());
            assert_eq!(result.candidate().arcs(), original.arcs());
            sketch = result.into_candidate();
        }
        sketch.set_position(entity(3), [0.0, -8.0]).unwrap();
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        let component = 8.0 * FRAC_1_SQRT_2;
        assert_point(
            result.candidate(),
            5,
            if counter_clockwise {
                [-component, component]
            } else {
                [component, -component]
            },
        );
        assert_eq!(result.candidate().arcs(), original.arcs());
    }
}
fn numbers(text: &str) -> Vec<f64> {
    text.split(',')
        .map(|value| value.parse().unwrap())
        .collect()
}
#[test]
fn read_only_sweep_midpoints_match_the_fixed_go_baseline() {
    for fixture in include_str!("../fixtures/go-arc-midpoints.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let input = numbers(fields[1]);
        let mut sketch = Sketch2D::default();
        for (index, coordinates) in input.as_chunks::<2>().0.iter().enumerate() {
            sketch
                .add_point(entity(index as u64 + 1), *coordinates)
                .unwrap();
        }
        sketch
            .add_arc(
                entity(4),
                entity(1),
                [entity(2), entity(3)],
                fields[2].parse().unwrap(),
            )
            .unwrap();
        let original = sketch.clone();
        let midpoint = sketch.arc_midpoint(entity(4)).unwrap();
        for (&actual, expected) in midpoint.iter().zip(numbers(fields[3])) {
            assert!(
                (actual - expected).abs() < 1e-12,
                "{}: {actual} vs {expected}",
                fields[0]
            );
        }
        assert_eq!(sketch, original);
    }
}
#[test]
fn midpoint_reference_errors_are_atomic_and_geometry_errors_keep_source_identity() {
    let mut sketch = midpoint_sketch(true, 1.0);
    let original = sketch.clone();
    for relation in [
        PlanarConstraint::ArcMidpoint {
            point: entity(999),
            arc: entity(4),
        },
        PlanarConstraint::ArcMidpoint {
            point: entity(5),
            arc: entity(999),
        },
    ] {
        assert!(
            sketch
                .add_constraint(constraint(99), relation)
                .unwrap_err()
                .to_string()
                .contains("999")
        );
    }
    assert!(
        sketch
            .arc_midpoint(entity(999))
            .unwrap_err()
            .to_string()
            .contains("999")
    );
    assert_eq!(sketch, original);
    sketch.set_position(entity(3), [4.0, 0.0]).unwrap();
    let query_error = sketch.arc_midpoint(entity(4)).unwrap_err();
    assert!(matches!(query_error, SketchError::ArcMidpoint { arc, .. } if arc == entity(4)));
    assert!(query_error.to_string().contains("ambiguous"));
    assert!(std::error::Error::source(&query_error).is_some());
    let error = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap_err();
    assert!(matches!(error, SketchError::Residual { constraint: id, .. } if id == constraint(5)));
    sketch
        .set_position(entity(2), [f64::MAX, f64::MAX])
        .unwrap();
    sketch
        .set_position(entity(1), [-f64::MAX, -f64::MAX])
        .unwrap();
    assert!(sketch.arc_midpoint(entity(4)).is_err());
}
#[test]
fn directed_midpoint_solve_rank_and_cancellation_preserve_working_units_and_input() {
    for scale in [1e-100, 1.0, 1e100] {
        let sketch = midpoint_sketch(true, scale);
        let mut parameters = parameters();
        parameters
            .set_expression(1, &format!("{} cm", 5.0 * scale))
            .unwrap();
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        let actual = result.candidate().point(entity(5)).unwrap().position;
        for component in actual {
            assert!((component / scale - 5.0 * FRAC_1_SQRT_2).abs() < 1e-8);
        }
        let cancelled = sketch
            .solve(&parameters, SolveOptions::default(), || true)
            .unwrap();
        assert_eq!(
            cancelled.numerical().termination,
            SolveTermination::Cancelled
        );
        assert_eq!(cancelled.candidate(), &sketch);
    }
}
