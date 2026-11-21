// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn rotations_scales_and_line_reversal_preserve_measurements_and_angle_units() {
    for scale in [1e-6, 1.0, 1e6] {
        for turn in [-1.2_f64, 0.0, 0.71] {
            for reverse in [false, true] {
                assert_transformed(scale, turn, reverse);
            }
        }
    }
}
fn assert_transformed(scale: f64, turn: f64, reverse: bool) {
    for (relation, expected) in rotation_invariant_cases() {
        let mut sketch = transformed_fixture(scale, turn, reverse);
        sketch
            .add_constraint(SketchConstraintId(17), relation)
            .unwrap();
        let measured = sketch.measure_dimension(SketchConstraintId(17)).unwrap();
        assert_scaled_quantity(measured, expected, scale);
    }
}
fn rotation_invariant_cases() -> Vec<(PlanarConstraint, Quantity)> {
    base_cases()
        .into_iter()
        .chain(angle_cases())
        .chain(offset_cases())
        .filter(|(relation, _)| {
            !matches!(
                relation,
                PlanarConstraint::Distance {
                    measure: DistanceMeasure::Horizontal | DistanceMeasure::Vertical,
                    ..
                }
            )
        })
        .collect()
}
fn transformed_fixture(scale: f64, turn: f64, reverse: bool) -> Sketch2D {
    let mut recipe = fixture().recipe();
    let (sin, cos) = turn.sin_cos();
    for point in &mut recipe.points {
        let [x, y] = point.position;
        point.position = [scale * (cos * x - sin * y), scale * (sin * x + cos * y)];
    }
    for circle in &mut recipe.circles {
        circle.radius *= scale;
    }
    if reverse {
        for line in &mut recipe.lines {
            line.endpoints.reverse();
        }
    }
    Sketch2D::from_recipe(&recipe).unwrap()
}
#[test]
fn zero_point_distance_remains_defined() {
    let relation = PlanarConstraint::Distance {
        points: [entity(1), entity(1)],
        measure: DistanceMeasure::Aligned,
        parameter: 99,
    };
    assert_measurement(relation, length(0.0));
}
#[test]
fn major_arc_measurement_retains_the_existing_full_sweep_model() {
    let mut recipe = fixture().recipe();
    recipe.arcs[0].counter_clockwise = true;
    let mut sketch = Sketch2D::from_recipe(&recipe).unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::ArcLength {
                arc: entity(7),
                parameter: 99,
            },
        )
        .unwrap();
    let measured = sketch.measure_dimension(SketchConstraintId(1)).unwrap();
    let expected = 6.0 * PI;
    assert!(
        (measured.value - expected).abs()
            <= expected * oblikovati_solve::nonlinear::SolveOptions::default().relative_tolerance
    );
}

fn assert_scaled_quantity(measured: Quantity, expected: Quantity, scale: f64) {
    let expected_value = if expected.dimension == Dimension::Length {
        expected.value * scale
    } else {
        expected.value
    };
    let tolerance = oblikovati_solve::nonlinear::SolveOptions::default().relative_tolerance
        * expected_value.abs().max(scale).max(1.0);
    assert!(
        (measured.value - expected_value).abs() <= tolerance,
        "{measured:?}: expected {expected_value}"
    );
    assert_eq!(measured.dimension, expected.dimension);
}
