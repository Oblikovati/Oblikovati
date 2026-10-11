// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
use oblikovati_solve::residual::{ResidualSource, ResidualTolerance};
fn source(layouts: [AxisLayout; 2]) -> PreparedConstraint {
    PreparedConstraint {
        projection: None,
        equation: PlanarEquation::AxisCollinear { layouts },
        variables: vec![],
    }
}
#[test]
fn collinear_derivatives_match_independent_differences_for_both_operand_orders() {
    let line = [1.0, 2.0, 4.0, 5.0];
    let anchors = [1.0, 2.0, 7.0, 8.0];
    let layouts = [AxisLayout::Line, AxisLayout::Ellipse { major_axis: false }];
    check_derivatives(&[anchors.as_slice(), &line, &[0.7]].concat(), layouts);
    let layouts = [layouts[1], layouts[0]];
    check_derivatives(
        &[&[7.0, 8.0, 1.0, 2.0][..], &[0.7][..], line.as_slice()].concat(),
        layouts,
    );
}
fn check_derivatives(values: &[f64], layouts: [AxisLayout; 2]) {
    let evaluated = source(layouts)
        .evaluate(&DualNumber::seed(values).unwrap())
        .unwrap();
    for column in 0..values.len() {
        let mut forward = values.to_vec();
        let mut backward = values.to_vec();
        forward[column] += 1e-5;
        backward[column] -= 1e-5;
        for (row, actual) in evaluated.iter().enumerate() {
            let difference =
                (scalar(&forward, layouts)[row] - scalar(&backward, layouts)[row]) / 2e-5;
            assert!((actual.gradient()[column] - difference).abs() < 1e-8);
        }
    }
}
fn scalar(values: &[f64], layouts: [AxisLayout; 2]) -> [f64; 2] {
    let first = angle(&values[4..4 + layouts[0].width()], layouts[0]);
    let second = angle(&values[4 + layouts[0].width()..], layouts[1]);
    let doubled = 2.0 * (second - first);
    let frame = if matches!(layouts[0], AxisLayout::Line) {
        second
    } else {
        first
    };
    [
        doubled.sin().atan2(doubled.cos()) * 0.5,
        frame.cos() * (values[3] - values[1]) - frame.sin() * (values[2] - values[0]),
    ]
}
fn angle(values: &[f64], layout: AxisLayout) -> f64 {
    match layout {
        AxisLayout::Line => (values[3] - values[1]).atan2(values[2] - values[0]),
        AxisLayout::Ellipse { major_axis } => {
            values[0]
                + if major_axis {
                    0.0
                } else {
                    std::f64::consts::FRAC_PI_2
                }
        }
    }
}
#[test]
fn line_only_collinear_offset_keeps_the_original_normalized_direction_arithmetic() {
    let layouts = [AxisLayout::Line; 2];
    let values = [0.0, 0.0, 7.0, 8.0, 0.0, 0.0, 3.0, 4.0, 7.0, 8.0, 10.0, 12.0];
    let actual = source(layouts)
        .evaluate(&DualNumber::seed(&values).unwrap())
        .unwrap();
    let expected = (3.0_f64 / 5.0) * 8.0 - (4.0_f64 / 5.0) * 7.0;
    assert_eq!(actual[1].value().to_bits(), expected.to_bits());
}
#[test]
fn mixed_domains_and_bad_variable_shapes_have_explicit_contracts() {
    let source = source([AxisLayout::Ellipse { major_axis: true }; 2]);
    assert_eq!(source.equations(), 2);
    assert_eq!(source.equation_tolerance(0), ResidualTolerance::Unitless);
    assert_eq!(source.equation_tolerance(1), ResidualTolerance::ModelLength);
    assert!(
        source
            .evaluate(&[])
            .unwrap_err()
            .to_string()
            .contains("expected 6")
    );
}
