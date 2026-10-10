// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{
    CircleContact, LineContactSide, SketchEntityId,
    residual::{PlanarEquation, PreparedConstraint},
};
use oblikovati_solve::residual::ResidualSource;
fn source(name: &str, scale: f64) -> (PreparedConstraint, Vec<f64>) {
    let arc = CircularRadius::Derived(SketchEntityId(1));
    let circle = CircularRadius::Independent(SketchEntityId(2));
    let (equation, prefix, frames, output, touch, input) = match name {
        "incidence" => (
            PlanarEquation::PointOnCircle,
            2,
            vec![arc],
            FrameOutput::Frames,
            false,
            vec![4.0, 6.0, 1.0, 2.0, 4.0, 6.0],
        ),
        "equal-radius" => (
            PlanarEquation::EqualRadius,
            0,
            vec![circle, arc],
            FrameOutput::Radii,
            false,
            vec![1.0, 2.0, 3.0, 1.0, 2.0, 4.0, 6.0],
        ),
        "radius" => (
            PlanarEquation::Radius(2.0 * scale),
            0,
            vec![arc],
            FrameOutput::Radii,
            false,
            vec![1.0, 2.0, 4.0, 6.0],
        ),
        "free-line" => (
            PlanarEquation::LineTangent(LineContactSide::Left),
            4,
            vec![arc],
            FrameOutput::Frames,
            false,
            vec![0.0, 0.0, 4.0, 0.0, 3.0, 2.0, 6.0, 6.0],
        ),
        "free-pair" => (
            PlanarEquation::CircleTangent(CircleContact::External),
            0,
            vec![circle, arc],
            FrameOutput::Frames,
            false,
            vec![0.0, 0.0, 2.0, 8.0, 1.0, 11.0, 5.0],
        ),
        "pinned-line" => (
            PlanarEquation::PinnedLineTangent {
                at_b: false,
                side: LineContactSide::Right,
            },
            4,
            vec![arc],
            FrameOutput::Frames,
            false,
            vec![0.0, 1.0, 5.0, 3.0, 0.0, 0.0, 0.0, 1.0],
        ),
        "pinned-pair" => (
            PlanarEquation::PinnedCircularTangent(CircleContact::External),
            0,
            vec![arc, arc],
            FrameOutput::Frames,
            true,
            vec![0.0, 0.0, 2.0, 0.0, 3.0, 0.5, 2.0, 0.0, 2.0, 0.0],
        ),
        _ => panic!("unexpected circular fixture {name}"),
    };
    let projection = CircularProjection {
        prefix,
        frames,
        output,
        touch,
    };
    assert_eq!(projection.input_count(), input.len());
    (
        PreparedConstraint {
            equation,
            variables: (0..input.len()).collect(),
            projection: Some(projection),
        },
        input.iter().map(|value| value * scale).collect(),
    )
}
fn scalar(name: &str, v: &[f64]) -> f64 {
    match name {
        "incidence" => (v[0] - v[2]).hypot(v[1] - v[3]) - (v[4] - v[2]).hypot(v[5] - v[3]),
        "equal-radius" => v[2] - (v[5] - v[3]).hypot(v[6] - v[4]),
        "radius" => (v[2] - v[0]).hypot(v[3] - v[1]) - 2.0,
        "free-line" | "pinned-line" => {
            let dx = v[2] - v[0];
            let dy = v[3] - v[1];
            let length = dx.hypot(dy);
            if name == "pinned-line" {
                return (dx * (v[4] - v[0]) + dy * (v[5] - v[1])) / length;
            }
            (dx * (v[5] - v[1]) - dy * (v[4] - v[0])) / length - (v[6] - v[4]).hypot(v[7] - v[5])
        }
        "free-pair" => (v[0] - v[3]).hypot(v[1] - v[4]) - v[2] - (v[5] - v[3]).hypot(v[6] - v[4]),
        "pinned-pair" => {
            ((v[8] - v[0]) * (v[5] - v[9]) - (v[9] - v[1]) * (v[4] - v[8]))
                / (v[8] - v[0]).hypot(v[9] - v[1])
        }
        _ => panic!("unexpected scalar fixture {name}"),
    }
}
#[test]
fn every_circular_projection_matches_go_raw_partials_and_independent_scalar_derivatives() {
    for fixture in include_str!("../fixtures/go-circular-frame-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let (source, input) = source(fields[0], 1.0);
        let actual = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
        assert_eq!(actual.len(), 1);
        assert!((actual[0].value() - fields[1].parse::<f64>().unwrap()).abs() < 1e-12);
        let expected: Vec<f64> = fields[2]
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        assert_eq!(actual[0].gradient().len(), expected.len());
        for (column, (&partial, expected)) in actual[0].gradient().iter().zip(expected).enumerate()
        {
            assert!(
                (partial - expected).abs() < 1e-12,
                "{} column {column}",
                fields[0]
            );
            let mut left = input.clone();
            let mut right = input.clone();
            left[column] -= 1e-5;
            right[column] += 1e-5;
            assert!(
                (partial - (scalar(fields[0], &right) - scalar(fields[0], &left)) / 2e-5).abs()
                    < 1e-8
            );
        }
    }
}
#[test]
fn projected_residual_units_and_derivatives_survive_extreme_coordinate_scales() {
    for name in [
        "incidence",
        "equal-radius",
        "radius",
        "free-line",
        "free-pair",
        "pinned-line",
        "pinned-pair",
    ] {
        let (reference, input) = source(name, 1.0);
        let reference = reference
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap();
        for scale in [1e-200, 1e-100, 1.0, 1e100, 1e200] {
            let (source, input) = source(name, scale);
            let actual = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
            assert!((actual[0].value() / scale - reference[0].value()).abs() < 1e-12);
            for (&actual, &expected) in actual[0].gradient().iter().zip(reference[0].gradient()) {
                assert!((actual - expected).abs() < 1e-12, "{name} scale {scale}");
            }
        }
    }
}
#[test]
fn projected_variable_shapes_and_contact_branch_domains_are_checked() {
    let (mut source, input) = source("pinned-line", 1.0);
    assert!(
        source
            .evaluate(&[])
            .unwrap_err()
            .to_string()
            .contains("expected 8")
    );
    source.equation = PlanarEquation::PinnedLineTangent {
        at_b: false,
        side: LineContactSide::Left,
    };
    assert!(
        source
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap_err()
            .to_string()
            .contains("signed distance")
    );
    let (mut source, input) = self::source("pinned-pair", 1.0);
    source.equation = PlanarEquation::PinnedCircularTangent(CircleContact::FirstContainsSecond);
    assert!(
        source
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap_err()
            .to_string()
            .contains("half-plane")
    );
    let mut aligned = input.clone();
    aligned[4] = 1.0;
    aligned[5] = 0.0;
    source.equation = PlanarEquation::PinnedCircularTangent(CircleContact::External);
    assert!(
        source
            .evaluate(&DualNumber::seed(&aligned).unwrap())
            .unwrap_err()
            .to_string()
            .contains("half-plane")
    );
}

fn circle_case(name: &str, scale: f64) -> (PreparedConstraint, PreparedConstraint, Vec<f64>) {
    let circle = CircularRadius::Independent(SketchEntityId(1));
    let (equation, prefix, frames, output, raw, canonical) = match name {
        "radius" => (
            PlanarEquation::Radius(2.0 * scale),
            0,
            vec![circle],
            FrameOutput::Radii,
            vec![1.0, 2.0, 3.0],
            vec![2],
        ),
        "equal" => (
            PlanarEquation::EqualRadius,
            0,
            vec![circle, circle],
            FrameOutput::Radii,
            vec![1.0, 2.0, 3.0, 4.0, 6.0, 2.0],
            vec![2, 5],
        ),
        "concentric" => (
            PlanarEquation::Coincident,
            0,
            vec![circle, circle],
            FrameOutput::Centers,
            vec![1.0, 2.0, 3.0, 4.0, 6.0, 2.0],
            vec![0, 1, 3, 4],
        ),
        "incidence" => (
            PlanarEquation::PointOnCircle,
            2,
            vec![circle],
            FrameOutput::Frames,
            vec![4.0, 6.0, 1.0, 2.0, 3.0],
            vec![0, 1, 2, 3, 4],
        ),
        "line" => (
            PlanarEquation::LineTangent(LineContactSide::Left),
            4,
            vec![circle],
            FrameOutput::Frames,
            vec![0.0, 0.0, 4.0, 0.0, 3.0, 2.0, 1.0],
            vec![0, 1, 2, 3, 4, 5, 6],
        ),
        "pair" => (
            PlanarEquation::CircleTangent(CircleContact::External),
            0,
            vec![circle, circle],
            FrameOutput::Frames,
            vec![0.0, 0.0, 3.0, 5.0, 1.0, 2.0],
            vec![0, 1, 3, 4, 2, 5],
        ),
        _ => panic!("unexpected circle fixture {name}"),
    };
    let projected = PreparedConstraint {
        equation,
        projection: Some(CircularProjection {
            prefix,
            frames,
            output,
            touch: false,
        }),
        variables: (0..raw.len()).collect(),
    };
    let canonical = PreparedConstraint {
        equation,
        projection: None,
        variables: canonical,
    };
    (
        projected,
        canonical,
        raw.iter().map(|value| value * scale).collect(),
    )
}
#[test]
fn independent_circle_values_and_global_partials_are_bit_identical_to_the_original_formulas() {
    for name in ["radius", "equal", "concentric", "incidence", "line", "pair"] {
        for scale in [1e-100, 1.0, 1e100] {
            let (projected, canonical, raw) = circle_case(name, scale);
            let global = DualNumber::seed(&raw).unwrap();
            let local: Vec<DualNumber> = canonical
                .variables
                .iter()
                .map(|&column| global[column].clone())
                .collect();
            let old = canonical.evaluate(&local).unwrap();
            let new = projected.evaluate(&global).unwrap();
            assert_eq!(new.len(), old.len());
            for (new, old) in new.iter().zip(old) {
                assert_eq!(
                    new.value().to_bits(),
                    old.value().to_bits(),
                    "{name} scale {scale}"
                );
                assert_eq!(
                    new.gradient()
                        .iter()
                        .map(|value| value.to_bits())
                        .collect::<Vec<_>>(),
                    old.gradient()
                        .iter()
                        .map(|value| value.to_bits())
                        .collect::<Vec<_>>(),
                    "{name} scale {scale}"
                );
            }
        }
    }
}

#[test]
fn diameter_rows_use_the_existing_radius_equation_with_half_the_go_residual_scale() {
    for fixture in include_str!("../fixtures/go-circular-dimensions.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        if !fields[0].starts_with("diameter-") {
            continue;
        }
        let input: Vec<f64> = fields[1]
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        let radius = if fields[0] == "diameter-circle" {
            CircularRadius::Independent(SketchEntityId(1))
        } else {
            CircularRadius::Derived(SketchEntityId(1))
        };
        let source = PreparedConstraint {
            equation: PlanarEquation::Radius(2.0),
            variables: (0..input.len()).collect(),
            projection: Some(CircularProjection {
                prefix: 0,
                frames: vec![radius],
                output: FrameOutput::Radii,
                touch: false,
            }),
        };
        let actual = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
        assert!((actual[0].value() * 2.0 - fields[2].parse::<f64>().unwrap()).abs() < 1e-12);
        let expected: Vec<f64> = fields[3]
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        assert_eq!(actual[0].gradient().len(), expected.len());
        for (&actual, expected) in actual[0].gradient().iter().zip(expected) {
            assert!((actual * 2.0 - expected).abs() < 1e-12);
        }
    }
}
