// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::axis_operand::AxisLayout;
#[test]
fn mixed_axis_pair_derivatives_match_independent_differences() {
    let layouts = [AxisLayout::Line, AxisLayout::Ellipse { major_axis: false }];
    check_derivatives(&[1.0, 2.0, 4.0, 5.0, 0.7], layouts);
    let layouts = [
        AxisLayout::Ellipse { major_axis: true },
        AxisLayout::Ellipse { major_axis: false },
    ];
    check_derivatives(&[0.2, 0.7], layouts);
}
fn check_derivatives(values: &[f64], layouts: [AxisLayout; 2]) {
    for perpendicular in [false, true] {
        let residual =
            pair_residual(&DualNumber::seed(values).unwrap(), layouts, perpendicular).unwrap();
        for column in 0..values.len() {
            let mut forward = values.to_vec();
            let mut backward = values.to_vec();
            forward[column] += 1e-5;
            backward[column] -= 1e-5;
            let difference = (scalar(&forward, layouts, perpendicular)
                - scalar(&backward, layouts, perpendicular))
                / 2e-5;
            assert!((residual[0].gradient()[column] - difference).abs() < 1e-8);
        }
    }
}
fn scalar(values: &[f64], layouts: [AxisLayout; 2], perpendicular: bool) -> f64 {
    let second = scalar_angle(&values[layouts[0].width()..], layouts[1]);
    let first = scalar_angle(&values[..layouts[0].width()], layouts[0]);
    let target = if perpendicular {
        std::f64::consts::FRAC_PI_2
    } else {
        0.0
    };
    let doubled = (second - first - target) * 2.0;
    doubled.sin().atan2(doubled.cos()) * 0.5
}
fn scalar_angle(values: &[f64], layout: AxisLayout) -> f64 {
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
