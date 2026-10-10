// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::SketchEntityId;
use crate::sketch::{
    circular_frame::CircularRadius,
    circular_projection::{CircularProjection, FrameOutput},
    residual::{PlanarEquation, PreparedConstraint},
};
use oblikovati_solve::{ad::DualNumber, residual::ResidualSource};
fn source(side: LineContactSide, target: f64, far_side: bool) -> PreparedConstraint {
    PreparedConstraint {
        projection: None,
        variables: (0..7).collect(),
        equation: PlanarEquation::TangentDistance {
            target,
            far_side,
            contact: TangentDistanceContact { side, lean: None },
            at_b: None,
        },
    }
}
fn scalar(v: &[f64; 7], far: bool, target: f64) -> f64 {
    let dx = v[2] - v[0];
    let dy = v[3] - v[1];
    let distance = ((dx * (v[5] - v[1]) - dy * (v[4] - v[0])) / dx.hypot(dy)).abs();
    distance + if far { v[6] } else { -v[6] } - target
}
#[test]
fn signed_offset_rows_match_independent_values_derivatives_and_extreme_working_units() {
    for side in [LineContactSide::Left, LineContactSide::Right] {
        for far in [false, true] {
            let input = [
                1.0,
                2.0,
                5.0,
                3.0,
                2.0,
                if side == LineContactSide::Left {
                    6.0
                } else {
                    -6.0
                },
                5.0,
            ];
            let reference = source(side, 2.0, far)
                .evaluate(&DualNumber::seed(&input).unwrap())
                .unwrap();
            assert!((reference[0].value() - scalar(&input, far, 2.0)).abs() < 1e-12);
            for column in 0..7 {
                let mut left = input;
                let mut right = input;
                left[column] -= 1e-5;
                right[column] += 1e-5;
                let partial = (scalar(&right, far, 2.0) - scalar(&left, far, 2.0)) / 2e-5;
                assert!((reference[0].gradient()[column] - partial).abs() < 1e-8);
            }
            for scale in [1e-200, 1e-100, 1.0, 1e100, 1e200] {
                let actual = source(side, 2.0 * scale, far)
                    .evaluate(&DualNumber::seed(&input.map(|value| value * scale)).unwrap())
                    .unwrap();
                assert!((actual[0].value() / scale - reference[0].value()).abs() < 1e-12);
                for (actual, expected) in actual[0].gradient().iter().zip(reference[0].gradient()) {
                    assert!((actual - expected).abs() < 1e-12);
                }
            }
        }
    }
}
fn numbers(text: &str) -> Vec<f64> {
    text.split(',')
        .map(|value| value.parse().unwrap())
        .collect()
}
#[test]
fn circle_and_arc_near_far_diameter_rows_match_go_fixtures() {
    for fixture in include_str!("../fixtures/go-tangent-distance-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let branch: Vec<&str> = fields[0].split(':').collect();
        let side = if branch[0] == "1" {
            LineContactSide::Left
        } else {
            LineContactSide::Right
        };
        let arc = branch[1] == "true";
        let far = branch[2] == "true";
        let diameter = branch[3] == "true";
        let factor = if diameter { 2.0 } else { 1.0 };
        let input = numbers(fields[1]);
        let mut source = source(side, 2.0 / factor, far);
        source.projection = Some(CircularProjection {
            prefix: 4,
            frames: vec![if arc {
                CircularRadius::Derived(SketchEntityId(2))
            } else {
                CircularRadius::Independent(SketchEntityId(2))
            }],
            output: FrameOutput::Frames,
            touch: false,
        });
        let actual = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
        assert!(
            (actual[0].value() * factor - numbers(fields[2])[0]).abs() < 1e-12,
            "{}",
            fields[0]
        );
        for (actual, expected) in actual[0].gradient().iter().zip(numbers(fields[3])) {
            assert!((actual * factor - expected).abs() < 1e-12);
        }
    }
}
#[test]
fn offset_domain_guards_refuse_collapsed_lines_wrong_sides_and_negative_radii() {
    for input in [
        [0.0, 0.0, 0.0, 0.0, 0.0, 2.0, 1.0],
        [0.0, 0.0, 4.0, 0.0, 0.0, -2.0, 1.0],
        [0.0, 0.0, 4.0, 0.0, 0.0, 2.0, 0.0],
    ] {
        assert!(
            source(LineContactSide::Left, 1.0, false)
                .evaluate(&DualNumber::seed(&input).unwrap())
                .is_err()
        );
    }
    assert!(
        source(LineContactSide::Left, 1.0, false)
            .evaluate(&[])
            .is_err()
    );
}
#[test]
fn shared_angular_row_matches_polar_derivatives_and_keeps_rank_at_exact_tangency() {
    for at_b in [false, true] {
        for side in [LineContactSide::Left, LineContactSide::Right] {
            let sign = if side == LineContactSide::Left {
                1.0
            } else {
                -1.0
            };
            let input = [
                0.0,
                0.0,
                4.0,
                0.0,
                if at_b { 4.0 } else { 0.0 },
                sign * 5.0,
                5.0,
            ];
            let source = PreparedConstraint {
                projection: None,
                variables: (0..7).collect(),
                equation: PlanarEquation::PinnedTangentDistance {
                    target: std::f64::consts::FRAC_PI_2,
                    at_b,
                    side,
                },
            };
            let actual = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
            assert!(actual[0].value().abs() < 1e-12);
            assert!((actual[0].gradient()[4].abs() - 0.2).abs() < 1e-12);
            assert_eq!(actual[0].gradient()[6], 0.0);
            let scalar = |v: &[f64; 7]| {
                let touch = if at_b { 2 } else { 0 };
                (v[5] - v[touch + 1]).atan2(v[4] - v[touch])
                    - (v[3] - v[1]).atan2(v[2] - v[0])
                    - sign * std::f64::consts::FRAC_PI_2
            };
            for column in 0..7 {
                let mut left = input;
                let mut right = input;
                left[column] -= 1e-5;
                right[column] += 1e-5;
                assert!(
                    (actual[0].gradient()[column] - (scalar(&right) - scalar(&left)) / 2e-5).abs()
                        < 1e-8
                );
            }
        }
    }
}
#[test]
fn shared_metric_without_radius_driver_retains_lean_and_uses_simple_tangent_root() {
    let contact = TangentDistanceContact {
        side: LineContactSide::Left,
        lean: Some(TangentLean::Positive),
    };
    let mut equation = PlanarEquation::TangentDistance {
        target: 0.0,
        far_side: false,
        contact,
        at_b: Some(false),
    };
    let input = [0.0, 0.0, 4.0, 0.0, 0.0, 5.0, 5.0];
    let source = |equation| PreparedConstraint {
        equation,
        projection: None,
        variables: (0..7).collect(),
    };
    let row = source(equation)
        .evaluate(&DualNumber::seed(&input).unwrap())
        .unwrap();
    assert_eq!(row[0].gradient()[4], 1.0);
    if let PlanarEquation::TangentDistance { ref mut target, .. } = equation {
        *target = -1.0;
    }
    let mut wrong = input;
    wrong[4] = -3.0;
    assert!(
        source(equation)
            .evaluate(&DualNumber::seed(&wrong).unwrap())
            .unwrap_err()
            .to_string()
            .contains("lean")
    );
    let mut opposite = contact;
    opposite.lean = Some(TangentLean::Negative);
    let source = source(PlanarEquation::TangentDistance {
        target: -1.0,
        far_side: false,
        contact: opposite,
        at_b: Some(false),
    });
    assert!(source.evaluate(&DualNumber::seed(&wrong).unwrap()).is_ok());
}
